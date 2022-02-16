package banners

func Granted() string {
	return `
  /$$$$$$                                 /$$                     /$$
 /$$__  $$                               | $$                    | $$
| $$  \__/  /$$$$$$  /$$$$$$  /$$$$$$$  /$$$$$$    /$$$$$$   /$$$$$$$
| $$ /$$$$ /$$__  $$|____  $$| $$__  $$|_  $$_/   /$$__  $$ /$$__  $$
| $$|_  $$| $$  \__/ /$$$$$$$| $$  \ $$  | $$    | $$$$$$$$| $$  | $$
| $$  \ $$| $$      /$$__  $$| $$  | $$  | $$ /$$| $$_____/| $$  | $$
|  $$$$$$/| $$     |  $$$$$$$| $$  | $$  |  $$$$/|  $$$$$$$|  $$$$$$$
\______/ |__/      \_______/|__/  |__/   \___/   \_______/ \_______/
																	
																	
`
}

func Assume() string {
	return `
  /$$$$$$                                                      
 /$$__  $$                                                     
| $$  \ $$  /$$$$$$$ /$$$$$$$ /$$   /$$ /$$$$$$/$$$$   /$$$$$$ 
| $$$$$$$$ /$$_____//$$_____/| $$  | $$| $$_  $$_  $$ /$$__  $$
| $$__  $$|  $$$$$$|  $$$$$$ | $$  | $$| $$ \ $$ \ $$| $$$$$$$$
| $$  | $$ \____  $$\____  $$| $$  | $$| $$ | $$ | $$| $$_____/
| $$  | $$ /$$$$$$$//$$$$$$$/|  $$$$$$/| $$ | $$ | $$|  $$$$$$$
|__/  |__/|_______/|_______/  \______/ |__/ |__/ |__/ \_______/
																  
																  
`
}

// cyan := color.New(color.FgHiCyan)
// cyan.Fprintf(os.Stderr, "\n%s\n", banners.Granted())

// magenta := color.New(color.FgHiMagenta)
// magenta.Fprintf(os.Stderr, "\n%s\n", banners.Assume())
