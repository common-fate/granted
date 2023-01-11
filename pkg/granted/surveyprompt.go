package granted

import (
	"errors"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/briandowns/spinner"
)

// code adapted from https://github.com/AlecAivazis/survey/blob/v2.3.5/select.go

const spinnerDelay = time.Millisecond * 100

/*
SelectNetwork is a prompt that presents a list of various options to the user
for them to select using the arrow keys and enter. Response type is a string.

	color := ""
	prompt := &surveyprompts.SelectNetwork{
		Message: "Choose a color:",
		Options: []string{"red", "blue", "green"},
	}
	survey.AskOne(prompt, &color)
*/
type SelectNetwork struct {
	survey.Renderer
	Message        string
	Options        func(filter string, page int) ([]string, int, error)
	Default        interface{}
	Help           string
	PageSize       int
	VimMode        bool
	FilterMessage  string
	Filter         func(filter string, value string, index int) bool
	Description    func(value string, index int) string
	filter         string //nolint:revive // nolint to keep code as close as possible from original
	selectedIndex  int
	useDefault     bool
	showingHelp    bool
	currentOptions []string
	loadedPages    int
	totalResults   int
	changingFilter bool
}

// SelectNetworkTemplateData is the data available to the templates when processing.
type SelectNetworkTemplateData struct {
	SelectNetwork
	PageEntries   []core.OptionAnswer
	SelectedIndex int
	Answer        string
	ShowAnswer    bool
	ShowHelp      bool
	Description   func(value string, index int) string
	Config        *survey.PromptConfig

	// These fields are used when rendering an individual option
	CurrentOpt   core.OptionAnswer
	CurrentIndex int
}

// IterateOption sets CurrentOpt and CurrentIndex appropriately so a select option can be rendered individually.
func (s SelectNetworkTemplateData) IterateOption(ix int, opt core.OptionAnswer) interface{} { //nolint:gocritic // survey expects a non pointer
	copy := s //nolint // nolint to keep code as close as possible from original
	copy.CurrentIndex = ix
	copy.CurrentOpt = opt
	return copy
}

func (s SelectNetworkTemplateData) GetDescription(opt core.OptionAnswer) string { //nolint:gocritic // survey expects a non pointer
	if s.Description == nil {
		return ""
	}
	return s.Description(opt.Value, opt.Index)
}

var SelectQuestionTemplate = `
{{- define "option"}}
    {{- if eq .SelectedIndex .CurrentIndex }}{{color .Config.Icons.SelectFocus.Format }}{{ .Config.Icons.SelectFocus.Text }} {{else}}{{color "default"}}  {{end}}
    {{- .CurrentOpt.Value}}{{ if ne ($.GetDescription .CurrentOpt) "" }} - {{color "cyan"}}{{ $.GetDescription .CurrentOpt }}{{end}}
    {{- color "reset"}}
{{end}}
{{- if .ShowHelp }}{{- color .Config.Icons.Help.Format }}{{ .Config.Icons.Help.Text }} {{ .Help }}{{color "reset"}}{{"\n"}}{{end}}
{{- color .Config.Icons.Question.Format }}{{ .Config.Icons.Question.Text }} {{color "reset"}}
{{- color "default+hb"}}{{ .Message }}{{ .FilterMessage }}{{color "reset"}}
{{- if .ShowAnswer}}{{color "cyan"}} {{.Answer}}{{color "reset"}}{{"\n"}}
{{- else}}
  {{- "  "}}{{- color "cyan"}}[Use arrows to move, type to filter{{- if and .Help (not .ShowHelp)}}, {{ .Config.HelpInput }} for more help{{end}}]{{color "reset"}}
  {{- "\n"}}
  {{- range $ix, $option := .PageEntries}}
    {{- template "option" $.IterateOption $ix $option}}
  {{- end}}
{{- end}}`

// OnChange is called on every keypress.
func (s *SelectNetwork) OnChange(key rune, config *survey.PromptConfig) (bool, error) { //nolint:gocyclo // nolint to keep code as close as possible from original
	options := core.OptionAnswerList(s.currentOptions)
	oldFilter := s.filter

	// if the user pressed the enter key and the index is a valid option
	if key == terminal.KeyEnter || key == '\n' { //nolint:gocritic // nolint to keep code as close as possible from original
		if !s.changingFilter {
			// if the selected index is a valid option
			if len(options) > 0 && s.selectedIndex < len(options) {
				// we're done (stop prompting the user)
				return true, nil
			}

			// we're not done (keep prompting)
			return false, nil
		}

		s.loadedPages = 0 // reset
		s.currentOptions = []string{}
		if err := s.loadNextPage(); err != nil {
			return false, err
		}
		options = core.OptionAnswerList(s.currentOptions)
		s.selectedIndex = 0
		// if the user pressed the up arrow or 'k' to emulate vim
	} else if (key == terminal.KeyArrowUp || (s.VimMode && key == 'k')) && len(options) > 0 {
		s.useDefault = false

		// if we are at the top of the list
		if s.selectedIndex > 0 {
			s.selectedIndex--
		}

		// if the user pressed down or 'j' to emulate vim
	} else if (key == terminal.KeyTab || key == terminal.KeyArrowDown || (s.VimMode && key == 'j')) && len(options) > 0 {
		s.useDefault = false
		// if we are at the bottom of the list
		s.selectedIndex++
		if s.selectedIndex >= len(options) {
			s.selectedIndex = len(options) - 1
			if s.shouldLoadMore() {
				if err := s.loadNextPage(); err != nil {
					return false, err
				}
				options = core.OptionAnswerList(s.currentOptions)
				s.selectedIndex++
			}
		}
		// only show the help message if we have one
	} else if string(key) == config.HelpInput && s.Help != "" {
		s.showingHelp = true
		// if the user wants to toggle vim mode on/off
	} else if key == terminal.KeyEscape {
		s.VimMode = !s.VimMode
		// if the user hits any of the keys that clear the filter
	} else if key == terminal.KeyDeleteWord || key == terminal.KeyDeleteLine {
		s.filter = ""
		// if the user is deleting a character in the filter
	} else if key == terminal.KeyDelete || key == terminal.KeyBackspace {
		// if there is content in the filter to delete
		if s.filter != "" {
			runeFilter := []rune(s.filter)
			// subtract a line from the current filter
			s.filter = string(runeFilter[0 : len(runeFilter)-1])
			// we removed the last value in the filter
		}
	} else if key >= terminal.KeySpace {
		s.filter += string(key)
		// make sure vim mode is disabled
		s.VimMode = false
		// make sure that we use the current value in the filtered list
		s.useDefault = false
	}

	s.FilterMessage = ""
	if s.filter != "" {
		s.FilterMessage = " " + s.filter
	}

	s.changingFilter = oldFilter != s.filter

	// figure out the options and index to render
	// figure out the page size
	pageSize := s.PageSize
	// if we dont have a specific one
	if pageSize == 0 {
		// grab the global value
		pageSize = config.PageSize
	}

	// TODO if we have started filtering and were looking at the end of a list
	// and we have modified the filter then we should move the page back!
	opts, idx := paginate(pageSize, options, s.selectedIndex)

	tmplData := SelectNetworkTemplateData{
		SelectNetwork: *s,
		SelectedIndex: idx,
		ShowHelp:      s.showingHelp,
		Description:   s.Description,
		PageEntries:   opts,
		Config:        config,
	}

	// render the options
	_ = s.RenderWithCursorOffset(SelectQuestionTemplate, tmplData, opts, idx)

	// keep prompting
	return false, nil
}

func (s *SelectNetwork) findSelectedIndex() (int, error) {
	if s.Default == "" || s.Default == nil {
		if err := s.loadNextPage(); err != nil { // load the first page
			return -1, err
		}
		return 0, nil
	}
	for {
		if !s.shouldLoadMore() { // keep loading pages until we have the option
			return 0, nil
		}
		if err := s.loadNextPage(); err != nil {
			return -1, err
		}
		for i, opt := range s.currentOptions {
			if opt == s.Default {
				return i, nil
			}
		}
	}
}

func (s *SelectNetwork) Prompt(config *survey.PromptConfig) (interface{}, error) { //nolint:gocyclo // nolint to keep code as close as possible from original
	// start off with the first option selected
	s.totalResults = -1
	sel, err := s.findSelectedIndex()
	if err != nil {
		return "", err
	}
	// save the selected index
	s.selectedIndex = sel

	// if there are no options to render
	if len(s.currentOptions) == 0 {
		// we failed
		return "", errors.New("please provide options to select from")
	}

	// figure out the page size
	pageSize := s.PageSize
	// if we dont have a specific one
	if pageSize == 0 {
		// grab the global value
		pageSize = config.PageSize
	}

	// figure out the options and index to render
	opts, idx := paginate(pageSize, core.OptionAnswerList(s.currentOptions), sel)

	cursor := s.NewCursor()
	cursor.Save()          //nolint:errcheck // for proper cursor placement during selection (nolint to keep code as close as possible from original)
	cursor.Hide()          //nolint:errcheck // hide the cursor (nolint to keep code as close as possible from original)
	defer cursor.Show()    //nolint:errcheck // show the cursor when we're done (nolint to keep code as close as possible from original)
	defer cursor.Restore() //nolint:errcheck // clear any accessibility offsetting on exit (nolint to keep code as close as possible from original)

	tmplData := SelectNetworkTemplateData{
		SelectNetwork: *s,
		SelectedIndex: idx,
		Description:   s.Description,
		ShowHelp:      s.showingHelp,
		PageEntries:   opts,
		Config:        config,
	}

	// ask the question
	err = s.RenderWithCursorOffset(SelectQuestionTemplate, tmplData, opts, idx)
	if err != nil {
		return "", err
	}

	// by default, use the default value
	s.useDefault = true

	rr := s.NewRuneReader()
	_ = rr.SetTermMode()
	defer func() {
		_ = rr.RestoreTermMode()
	}()

	// start waiting for input
	for {
		r, _, err := rr.ReadRune() //nolint:govet // nolint to keep code as close as possible from original
		if err != nil {
			return "", err
		}
		if r == terminal.KeyInterrupt {
			return "", terminal.InterruptErr
		}
		if r == terminal.KeyEndTransmission {
			break
		}
		done, err := s.OnChange(r, config)
		if err != nil {
			return "", err
		}
		if done {
			break
		}
	}
	options := core.OptionAnswerList(s.currentOptions)
	s.filter = ""
	s.FilterMessage = ""

	// the index to report
	var val string
	// if we are supposed to use the default value
	if s.useDefault || s.selectedIndex >= len(options) {
		// if there is a default value
		if s.Default != nil {
			// if the default is a string
			if defaultString, ok := s.Default.(string); ok { //nolint:gocritic // nolint to keep code as close as possible from original
				// use the default value
				val = defaultString
				// the default value could also be an interpret which is interpretted as the index
			} else if defaultIndex, ok := s.Default.(int); ok { //nolint:revive // nolint to keep code as close as possible from original
				val = s.currentOptions[defaultIndex]
			} else {
				return val, errors.New("default value of select must be an int or string")
			}
		} else if len(options) > 0 {
			// there is no default value so use the first
			val = options[0].Value
		}
		// otherwise the selected index points to the value
	} else if s.selectedIndex < len(options) {
		// the
		val = options[s.selectedIndex].Value
	}

	// now that we have the value lets go hunt down the right index to return
	idx = -1
	for i, optionValue := range s.currentOptions {
		if optionValue == val {
			idx = i
		}
	}

	return core.OptionAnswer{Value: val, Index: idx}, err
}

func (s *SelectNetwork) Cleanup(config *survey.PromptConfig, val interface{}) error {
	cursor := s.NewCursor()
	cursor.Restore() //nolint:errcheck // nolint to keep code as close as possible from original
	return s.Render(
		SelectQuestionTemplate,
		SelectNetworkTemplateData{
			SelectNetwork: *s,
			Answer:        val.(core.OptionAnswer).Value,
			ShowAnswer:    true,
			Description:   s.Description,
			Config:        config,
		},
	)
}

func (s *SelectNetwork) shouldLoadMore() bool {
	return s.totalResults == -1 || len(s.currentOptions) < s.totalResults
}

func (s *SelectNetwork) loadNextPage() error {
	spin := spinner.New(spinner.CharSets[9], spinnerDelay)
	spin.Start()

	opt, results, err := s.Options(s.filter, s.loadedPages)
	spin.Stop()
	if err != nil {
		return err
	}

	s.currentOptions = append(s.currentOptions, opt...)
	s.totalResults = results
	s.loadedPages++
	return nil
}

// code adapted from https://github.com/AlecAivazis/survey/blob/v2.3.5/survey.go#L371

func paginate(pageSize int, choices []core.OptionAnswer, sel int) ([]core.OptionAnswer, int) { //nolint:gocritic // nolint to keep code as close as possible from original
	var start, end, cursor int

	if len(choices) < pageSize { //nolint:gocritic // nolint to keep code as close as possible from original
		// if we dont have enough options to fill a page
		start = 0
		end = len(choices)
		cursor = sel
	} else if sel < pageSize/2 {
		// if we are in the first half page
		start = 0
		end = pageSize
		cursor = sel
	} else if len(choices)-sel-1 < pageSize/2 {
		// if we are in the last half page
		start = len(choices) - pageSize
		end = len(choices)
		cursor = sel - start
	} else {
		// somewhere in the middle
		above := pageSize / 2 //nolint:gomnd // nolint to keep code as close as possible from original
		below := pageSize - above

		cursor = pageSize / 2 //nolint:gomnd // nolint to keep code as close as possible from original
		start = sel - above
		end = sel + below
	}

	// return the subset we care about and the index
	return choices[start:end], cursor
}
