#compdef assume
_cli_zsh_autocomplete() {

  local -a opts
  local cur
  cur=${words[-1]}

  # adapted from https://github.com/urfave/cli/blob/main/autocomplete/zsh_autocomplete  
  # Breakdown of the shell script
  # '(@f)' split the results of the call to assumego into an array, slit at newlines
  # '_CLI_ZSH_AUTOCOMPLETE_HACK=1' something required by urfavcli
  # 'FORCE_NO_ALIAS=true assumego' call the assumego binary to avoid shell script interference with auto complete
  # '${words[@]:1:#words[@]-1}' slice the arguments (words[@]) starting at index 1 get (number of arguments - 1)
  # '--generate-bash-completion' tell urfavcli to run autocomplete
  # not sure what cur does exactly, its from the urfav example something to do with cli flags
  if [[ "$cur" == "-"* ]]; then
    opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 FORCE_NO_ALIAS=true assumego ${words[@]:1:#words[@]-1} ${cur} --generate-bash-completion)}")
  else
    opts=("${(@f)$(_CLI_ZSH_AUTOCOMPLETE_HACK=1 FORCE_NO_ALIAS=true assumego ${words[@]:1:#words[@]-1} --generate-bash-completion)}")
  fi

 
  # if autocomplete is available, print it else show files in the directory as options
  if [[ "${opts[1]}" != "" ]]; then
    _describe 'values' opts
  else
    _files
  fi

  return
}

compdef _cli_zsh_autocomplete assume