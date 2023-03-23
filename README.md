# gomenutree
Simple CLI menu to map nested menu options to functions

# Usage
* Import the package <br />
  `import "github.com/mikefrom1974/gomenutree"`
* Create your menu(s) <br />
  `mMain := gomenutree.NewMenu("Main", "myPrompt")`
* Add options -> functions <br />
  `mMain.AddOption("foo", foo)`
* Create your tree and add menus <br />
  `mTree := gomenutree.NewMenuTree(mMain)` <br />
  `mTree.AddSubMenu(<parentMenu>, <childMenu>)`
* Display your menu<br />
  `mTree.Display()`
* Optionally set the current menu prompt <br />
  `mTree.SetPrompt("Please select one of the following:")`

# Notes
* For simplicity, mapped functions are without parameters 
  (to avoid interfaces and reflections, etc). The user is
  expected to handle persistent values on their own.
  If you must have parameters, use a function wrapper
  (see example).

# Sample:
```go
package main

import (
	"fmt"
	
	"github.com/mikefrom1974/gomenutree"
)

func main() {
	prompt := "Please select from the following:"
	
	mMain := gomenutree.NewMenu("test main", prompt)
	mMain.AddOption("foo", foo)
	mMain.AddOption("bar", bar)

	mTree := gomenutree.NewMenuTree(mMain)

	mSub1 := gomenutree.NewMenu("simple sub", "")
	mSub1.AddOption("baz", baz)
	mTree.AddSubMenu(mMain, mSub1)

	mSub2 := gomenutree.NewMenu("func w param", prompt)
	mSub2.AddOption("enter", func() {
		p("staticParam")
	})
	mSub2.AddOption("examine", func() {
		p(mTree.Name() + " " + mTree.Prompt())
	})
	mTree.AddSubMenu(mMain, mSub2)

	mTree.Display()
}

func foo() {
	fmt.Println("foo")
}

func bar() {
	fmt.Println("bar")
}

func baz() {
	fmt.Println("baz")
}

func p(param string) {
	fmt.Println("parameter:", param)
}
```

# Change Log _(semantic versioning)_
**1.0.0**
* *Added*: Initial release

**1.0.1**
* *Fixed*: incomplete line fill on redraw

**1.0.2**
* *Fixed*: typos

**1.0.3**
* *Changed*: selecting by hotkey now moves selection cursor

* **1.0.3**
* *Fixed*: redraw scrambles on small terminals (added toggle)
