package a

// --- should detect ---

func assignBeforeIf() {
	x := 1

	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

func assignBeforeFor() {
	items := []int{1, 2, 3}

	for _, item := range items { // want `unnecessary blank line before block using items`
		println(item)
	}
}

func assignBeforeSwitch() {
	mode := "json"

	switch mode { // want `unnecessary blank line before block using mode`
	case "json":
		println("json")
	}
}

func assignBeforeGoFunc() {
	ch := make(chan int)

	go func() { // want `unnecessary blank line before block using ch`
		ch <- 1
	}()
}

func assignBeforeDeferFunc() {
	cleanup := func() {}

	defer func() { // want `unnecessary blank line before block using cleanup`
		cleanup()
	}()
}

func varDecl() {
	var x int

	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

func multiAssign() {
	x, y := 1, 2

	if y > 0 { // want `unnecessary blank line before block using y`
		println(y)
	}
	_ = x
}

func multipleBlankLines() {
	x := 1

	if x > 0 { // want `unnecessary blank line before block using x`
		println(x)
	}
}

// --- should NOT detect ---

func unusedInBlock() {
	x := 1

	if true {
		println("hello")
	}
	_ = x
}

func alreadyCuddled() {
	x := 1
	if x > 0 {
		println(x)
	}
}

func commentBetween() {
	x := 1
	// this is intentional
	if x > 0 {
		println(x)
	}
}

func notAssignment() {
	println("hello")

	if true {
		println("world")
	}
}

func goPlainCall() {
	ch := make(chan int)

	go println(ch)
}

func blankIdentifier() {
	_ = 1

	if true {
		println("hello")
	}
}
