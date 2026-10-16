// Package mock holds hand-written fakes of domain interfaces and app ports.
//
// Each mock exposes a function field per method and an Invoked flag for tests.
// These fakes are handwritten; no mock generator or library is used.
package mock

import app "xorbit/internal/app"

// Terminal is a fake application.Terminal.
type Terminal struct {
	DrawFn      func(frame []byte) error
	DrawInvoked bool

	ReadLineFn      func(dst []byte) (int, error)
	ReadLineInvoked bool
}

func (t *Terminal) Draw(frame []byte) error {
	t.DrawInvoked = true
	return t.DrawFn(frame)
}

func (t *Terminal) ReadLine(dst []byte) (int, error) {
	t.ReadLineInvoked = true
	return t.ReadLineFn(dst)
}

// Display is a fake application.Display.
type Display struct {
	ShowFn      func(result []byte) error
	ShowInvoked bool
}

func (d *Display) Show(result []byte) error {
	d.ShowInvoked = true
	return d.ShowFn(result)
}

// Allocator is a fake application.Allocator.
type Allocator struct {
	AllocFn      func(n int) ([]byte, error)
	AllocInvoked bool
}

func (a *Allocator) Alloc(n int) ([]byte, error) {
	a.AllocInvoked = true
	return a.AllocFn(n)
}

// Dice is a fake application.DiceSource. View defaults to an empty dice
// state, and other methods default to no-op behavior.
type Dice struct {
	StartFn      func(scratch []byte) error
	StartInvoked bool
	StartCalls   int
	StartScratch []byte

	ViewFn      func() app.DiceView
	ViewInvoked bool

	InputFn      func(line []byte) app.DiceStep
	InputInvoked bool
	Inputs       [][]byte
	InputSteps   []app.DiceStep

	FinishFn      func(dst []byte) error
	FinishInvoked bool
	FinishCalls   int
	FinishDst     []byte

	// Hooks to record behaviour the App observes.
	StartErr  error
	FinishErr error
}

func (d *Dice) View() app.DiceView {
	d.ViewInvoked = true
	if d.ViewFn != nil {
		return d.ViewFn()
	}
	return app.DiceView{
		Window: make([]byte, app.DiceWindowSize),
		Digest: make([]byte, 32),
	}
}

func (d *Dice) Start(scratch []byte) error {
	d.StartInvoked = true
	d.StartCalls++
	d.StartScratch = scratch
	if d.StartFn != nil {
		return d.StartFn(scratch)
	}
	return d.StartErr
}

func (d *Dice) Input(line []byte) app.DiceStep {
	d.InputInvoked = true
	// Copy the line because the caller reuses the buffer.
	cp := append([]byte(nil), line...)
	d.Inputs = append(d.Inputs, cp)
	var step app.DiceStep
	if d.InputFn != nil {
		step = d.InputFn(line)
	}
	d.InputSteps = append(d.InputSteps, step)
	return step
}

func (d *Dice) Finish(dst []byte) error {
	d.FinishInvoked = true
	d.FinishCalls++
	d.FinishDst = dst
	if d.FinishFn != nil {
		return d.FinishFn(dst)
	}
	return d.FinishErr
}

// Kernel is a fake application.KernelRNG.
type Kernel struct {
	GenerateFn      func(dst []byte) error
	GenerateInvoked bool
	GenerateCalls   int
	GenerateDst     []byte
	GenerateErr     error
}

func (k *Kernel) Generate(dst []byte) error {
	k.GenerateInvoked = true
	k.GenerateCalls++
	k.GenerateDst = dst
	if k.GenerateFn != nil {
		return k.GenerateFn(dst)
	}
	return k.GenerateErr
}
