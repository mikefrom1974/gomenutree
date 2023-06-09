package gomenutree

import (
	"fmt"
	"strings"

	"github.com/pkg/term"
	"github.com/ttacon/chalk"
)

type (
	// MenuTree serves as the main structure, holding menus and submenus along with configuration
	MenuTree struct {
		homeMenu     *Menu
		currentMenu  *Menu
		previousMenu *Menu
		subMenuMap   map[*Menu][]*Menu
		displaying   bool

		Redraw bool //whether to back up and redraw the menu in place
	}

	// Menu struct holds the map of options to functions, as well as configuration
	Menu struct {
		name            string
		prompt          string
		promptFunction  func() string
		options         map[string]func()
		optionsOrder    []string
		selection       int
		hotKeys         map[string]int
		lastRenderLines int
		longestLine     int
	}
)

const (
	up       byte = 65 // arrow keys are the last byte of a 3 byte sequence
	down     byte = 66
	left     byte = 68
	right    byte = 67
	escape   byte = 27
	enter    byte = 13
	backtick byte = 96
	exitX    byte = 120
	ctrlC    byte = 3

	upDownArrow = '\u2195'
	leftArrow   = '\u2190'
	rightArrow  = '\u2192'
)

// NewMenuTree will create and return a new go menu tree. This will be the main object used by the user.
func NewMenuTree(homeMenu *Menu) *MenuTree {
	m := new(MenuTree)
	m.homeMenu = homeMenu
	m.currentMenu = homeMenu
	m.Redraw = true
	m.subMenuMap = make(map[*Menu][]*Menu)
	return m
}

// NewMenu will create a new Menu object (to be used as the main menu in NewMenuTree or as a submenu).
// prompt and promptFunction are mutually exclusive with promptFunction taking priority if not nil
// prompt function must take no parameters and return a string
func NewMenu(name string, prompt string, promptFunction func() string) *Menu {
	m := new(Menu)
	m.name = name
	if promptFunction != nil {
		m.prompt = ""
		m.promptFunction = promptFunction
	} else {
		m.prompt = prompt
		m.promptFunction = nil
	}
	m.options = make(map[string]func())
	return m
}

// Name will return the name of the current menu (not exposed since menu names can not be changed)
func (m *MenuTree) Name() string {
	return m.currentMenu.name
}

// Prompt will return the prompt for the current menu
func (m *MenuTree) Prompt() string {
	return m.currentMenu.prompt
}

// SetPrompt will set a static text prompt, or a function to generate the prompt on render, for the current menu
// prompt and promptFunction are mutually exclusive with promptFunction taking priority if not nil
// prompt function must take no parameters and return a string
func (m *MenuTree) SetPrompt(prompt string, promptFunction func() string) {
	if promptFunction != nil {
		m.currentMenu.prompt = ""
		m.currentMenu.promptFunction = promptFunction
	} else {
		m.currentMenu.prompt = prompt
		m.currentMenu.promptFunction = nil
	}
}

// AddOption will add a named option to the list of menu selections, mapped to a function
func (m *Menu) AddOption(name string, function func()) {
	m.options[name] = function
	for i, n := range m.optionsOrder {
		if n == name {
			m.optionsOrder = append(m.optionsOrder[:i], m.optionsOrder[i+1:]...)
		}
	}
	m.optionsOrder = append(m.optionsOrder, name)
}

// DeleteOption will remove an option from the list of menu selections
func (m *Menu) DeleteOption(name string) {
	delete(m.options, name)
	for i, n := range m.optionsOrder {
		if n == name {
			m.optionsOrder = append(m.optionsOrder[:i], m.optionsOrder[i+1:]...)
		}
	}
}

// AddSubMenu will add the child menu to the list of submenu selections in the parent menu
func (m *MenuTree) AddSubMenu(parentMenu *Menu, childMenu *Menu) {
	if _, ok := m.subMenuMap[parentMenu]; !ok {
		m.subMenuMap[parentMenu] = []*Menu{childMenu}
	} else {
		m.subMenuMap[parentMenu] = append(m.subMenuMap[parentMenu], childMenu)
	}
}

// AddSubMenus will add a slice of menus to the list of submenu selections in the parent menu
func (m *MenuTree) AddSubMenus(parentMenu *Menu, childMenus []*Menu) {
	if _, ok := m.subMenuMap[parentMenu]; !ok {
		m.subMenuMap[parentMenu] = childMenus
	} else {
		m.subMenuMap[parentMenu] = append(m.subMenuMap[parentMenu], childMenus...)
	}
}

// DeleteSubMenu will remove the menu from the list of submenu selections in the parent menu
func (m *MenuTree) DeleteSubMenu(parentMenu *Menu, childMenu *Menu) {
	if _, ok := m.subMenuMap[parentMenu]; ok {
		for i, sm := range m.subMenuMap[parentMenu] {
			if sm == childMenu {
				m.subMenuMap[parentMenu] = append(m.subMenuMap[parentMenu][:i], m.subMenuMap[parentMenu][i+1:]...)
				break
			}
		}
	}
}

// ChangeMenu will jump straight to the given menu, setting the current menu to the "back" action result
func (m *MenuTree) ChangeMenu(menu *Menu) {
	m.previousMenu = m.currentMenu
	if menu == m.homeMenu {
		m.previousMenu = nil
	}
	m.currentMenu = menu
	m.currentMenu.lastRenderLines = 0
	if m.displaying {
		m.render()
	}
}

// render will draw the current menu, optionally redrawing (erasing and writing over itself)
func (m *MenuTree) render() {
	if m.currentMenu.lastRenderLines > 0 && m.Redraw {
		fmt.Printf("\033[%dA", m.currentMenu.lastRenderLines)
	}
	var lines []string
	m.currentMenu.hotKeys = make(map[string]int)
	lines = append(lines, fmt.Sprintf("Menu: %s", chalk.Bold.TextStyle(m.currentMenu.name)))
	if m.currentMenu.promptFunction != nil {
		m.currentMenu.prompt = m.currentMenu.promptFunction()
	}
	if m.currentMenu.prompt != "" {
		m.currentMenu.prompt = strings.Replace(m.currentMenu.prompt, "\r\n", "\n", -1)
		m.currentMenu.prompt = strings.Replace(m.currentMenu.prompt, "\n\r", "\n", -1)
		promptLines := strings.Split(m.currentMenu.prompt, "\n")
		for _, l := range promptLines {
			lines = append(lines, fmt.Sprintf(" %v", l))
		}
	}
	for i, o := range m.currentMenu.optionsOrder {
		if i == 0 {
			lines = append(lines, fmt.Sprintf("%s", chalk.Bold.TextStyle("Options:")))
		}
		if hk := m.currentMenu.assignHotkey(o, i); hk != "" {
			o = strings.Replace(o, hk, chalk.Underline.TextStyle(hk), 1)
		}
		if i == m.currentMenu.selection {
			lines = append(lines, fmt.Sprintf(">%s", chalk.Italic.TextStyle(o)))
		} else {
			lines = append(lines, fmt.Sprintf(" %s", o))
		}
	}
	if smm, ok := m.subMenuMap[m.currentMenu]; ok {
		lines = append(lines, fmt.Sprintf("%s", chalk.Bold.TextStyle("SubMenus:")))
		for i, sm := range smm {
			mIdx := i + len(m.currentMenu.optionsOrder)
			line := sm.name
			if hk := m.currentMenu.assignHotkey(line, mIdx); hk != "" {
				line = strings.Replace(line, hk, chalk.Underline.TextStyle(hk), 1)
			}
			if mIdx == m.currentMenu.selection {
				lines = append(lines, fmt.Sprintf(">%s", chalk.Italic.TextStyle(line)))
			} else {
				lines = append(lines, fmt.Sprintf(" %s", line))
			}
		}
	}
	lines = append(lines, "")
	if m.previousMenu != nil {
		lines = append(lines, fmt.Sprintf(" %c/esc back to %s, E%sit ", leftArrow, m.previousMenu.name, chalk.Underline.TextStyle("x")))
	} else {
		lines = append(lines, fmt.Sprintf("E%sit", chalk.Underline.TextStyle("x")))
	}
	m.currentMenu.longestLine = 0
	for _, l := range lines {
		if len(l) > m.currentMenu.longestLine {
			m.currentMenu.longestLine = len(l)
		}
	}
	m.currentMenu.longestLine += 2
	m.currentMenu.lastRenderLines = len(lines) + 1
	header := "\n"
	for i := 0; i < m.currentMenu.longestLine+4; i++ {
		header += "*"
	}
	fmt.Println(header)
	for idx, l := range lines {
		fillLength := m.currentMenu.longestLine - len(l)
		if idx < len(lines)-1 {
			l = "  " + l + "\n"
			fmt.Print(l)
		} else {
			l = "**" + l
			for i := 0; i < fillLength; i++ {
				l += "*"
			}
			l += "**"
			fmt.Print(l)
		}
	}
}

// Display will initiate the menu tree (after initial config) and render the current menu
func (m *MenuTree) Display() {
	m.displaying = true
	m.currentMenu.selection = 0
	defer func() {
		fmt.Printf("\033[?25h")
	}()
	redrawPrevious := m.Redraw
	m.Redraw = false
	fmt.Println("Welcome to go menu tree.")
	fmt.Printf("%c to move selection cursor.\n", upDownArrow)
	fmt.Printf("%c/Enter/H%stkey to choose.\n", rightArrow, chalk.Underline.TextStyle("o"))
	fmt.Printf("%c/Esc to go back, %s to Exit.\n", leftArrow, chalk.Underline.TextStyle("x"))
	fmt.Println("` (backtick) to toggle redraw (small terminals may scramble)")
	fmt.Println("Press any key to start menu...")
	m.getInput()
	m.render()
	m.Redraw = redrawPrevious
	fmt.Printf("\033[?25l")
	for m.displaying {
		input := strings.ToUpper(m.getInput())
		switch input {
		case "UP":
			m.currentMenu.selection -= 1
			if m.currentMenu.selection < 0 {
				m.currentMenu.selection = len(m.currentMenu.optionsOrder) - 1
				if smm, ok := m.subMenuMap[m.currentMenu]; ok {
					m.currentMenu.selection += len(smm)
				}
			}
			m.render()
		case "DOWN":
			m.currentMenu.selection += 1
			total := len(m.currentMenu.optionsOrder) - 1
			if smm, ok := m.subMenuMap[m.currentMenu]; ok {
				total += len(smm)
			}
			if m.currentMenu.selection > total {
				m.currentMenu.selection = 0
			}
			m.render()
		case "ENTER":
			m.execute(m.currentMenu.selection)
		case "BACK":
			if m.previousMenu != nil {
				m.ChangeMenu(m.previousMenu)
			}
		case "TOGGLE":
			if m.Redraw {
				m.Redraw = false
				fmt.Println("\nredraw disabled")
				m.render()
			} else {
				fmt.Println("\nredraw enabled")
				m.render()
				m.Redraw = true
			}
		case "":
		//do nothing
		case "EXIT":
			if i, ok := m.currentMenu.hotKeys[input]; !ok {
				m.displaying = false
			} else {
				m.execute(i)
			}
		default:
			if i, ok := m.currentMenu.hotKeys[input]; ok {
				m.currentMenu.selection = i
				m.execute(i)
			}
		}
	}
	fmt.Println()
}

// execute will act on an option > function selection or go into a submenu, depending on selection
func (m *MenuTree) execute(index int) {
	if index >= 0 && index < len(m.currentMenu.optionsOrder) {
		if m.Redraw {
			fmt.Printf("\033[%dA", 2)
		}
		m.currentMenu.lastRenderLines = 0
		fName := m.currentMenu.optionsOrder[index]
		line := fmt.Sprintf("\n*** Executing %s... ***", fName)
		fill := m.currentMenu.longestLine - len(line)
		if fill > 0 {
			for i := 0; i < fill; i++ {
				line += "*"
			}
		}
		fmt.Println(line)
		if f, ok := m.currentMenu.options[fName]; ok {
			line = "------------- Output -------------"
			fill = m.currentMenu.longestLine - len(line)
			if fill > 0 {
				for i := 0; i < fill; i++ {
					line += "-"
				}
			}
			fmt.Println(line)
			f()
			line = "-------------- End ---------------"
			fill = m.currentMenu.longestLine - len(line)
			if fill > 0 {
				for i := 0; i < fill; i++ {
					line += "-"
				}
			}
			fmt.Println(line)
			fmt.Println("(Press any key to continue)")
			m.getInput()
			fmt.Println()
			m.render()
		} else {
			fmt.Println("\nError, function not found in Options map.")
			fmt.Println("(Press any key to continue)")
		}
	} else {
		subIndex := index - len(m.currentMenu.optionsOrder)
		if smm, ok := m.subMenuMap[m.currentMenu]; !ok {
			fmt.Println("\nError, menu not found in subMenu map.")
			fmt.Println("(Press any key to continue)")
			m.currentMenu.lastRenderLines += 2
			m.getInput()
			m.render()
		} else {
			if subIndex >= 0 && subIndex < len(smm) {
				m.ChangeMenu(smm[subIndex])
			} else {
				fmt.Println("\nError, function not found in Options map.")
				fmt.Println("(Press any key to continue)")
				m.currentMenu.lastRenderLines += 2
				m.getInput()
				m.render()
			}
		}
	}
}

// assignHotKey handles auto-creating hotkeys for named entries, while avoiding duplication
func (m *Menu) assignHotkey(name string, index int) (hotkey string) {
	for _, ch := range strings.Split(name, "") {
		uch := strings.ToUpper(ch)
		if uch == "X" {
			continue
		}
		if _, ok := m.hotKeys[uch]; !ok {
			m.hotKeys[uch] = index
			return ch
		}
	}
	return ""
}

// getInput will listen for a single keystroke (for navigating the menu)
func (m *MenuTree) getInput() string {
	tty, tErr := term.Open("/dev/tty")
	if tErr != nil {
		panic(tErr)
	}
	defer func() {
		_ = tty.Restore()
		_ = tty.Close()
	}()
	if e := term.RawMode(tty); e != nil {
		panic(e)
	}
	bb := make([]byte, 3)
	if n, e := tty.Read(bb); e != nil {
		panic(e)
	} else {
		if n == 3 {
			switch bb[2] {
			case up:
				return "UP"
			case down:
				return "DOWN"
			case left:
				return "BACK"
			case right:
				return "ENTER"
			default:
				return "DOWN"
			}
		} else {
			switch bb[0] {
			case enter:
				return "ENTER"
			case escape:
				return "BACK"
			case backtick:
				return "TOGGLE"
			case exitX, ctrlC:
				return "EXIT"
			default:
				return string(bb[0])
			}
		}
	}
	return ""
}
