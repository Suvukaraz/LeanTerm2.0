package gtk

import (
	"image"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
)

// tooltipDelay is how long the pointer rests on a widget before its
// tooltip shows (Qt and GTK use about this).
const tooltipDelay = 600 * time.Millisecond

// tooltips is the tooltip state of a window's theme: widgets report their
// hover with Tip during the frame; LayoutTooltip draws the one that is due.
type tooltips struct {
	since map[any]time.Time // hovered since (zero time: suppressed until left)
	seen  map[any]bool      // reported in this frame
	text  string
}

// Tip shows text as the tooltip of a widget (key identifies it, e.g. its
// Clickable) once the pointer has rested on it for a moment. pressed hides
// it until the pointer leaves the widget.
func (th *Theme) Tip(gtx layout.Context, key any, hovered, pressed bool, text string) {
	t := &th.tips
	if t.since == nil {
		t.since, t.seen = map[any]time.Time{}, map[any]bool{}
	}
	if !hovered || text == "" {
		delete(t.since, key)
		return
	}
	t.seen[key] = true
	start, ok := t.since[key]
	switch {
	case pressed:
		t.since[key] = time.Time{}
		return
	case !ok:
		start = gtx.Now
		t.since[key] = start
	case start.IsZero():
		return // clicked: not again until the pointer leaves
	}
	if due := start.Add(tooltipDelay); gtx.Now.Before(due) {
		gtx.Execute(op.InvalidateCmd{At: due})
		return
	}
	t.text = text
}

// ClickableTip is Tip for a Clickable.
func (th *Theme) ClickableTip(gtx layout.Context, c *widget.Clickable, text string) {
	th.Tip(gtx, c, c.Hovered(), c.Pressed(), text)
}

// LayoutTooltip draws the tooltip that is due near the pointer (ptr, in the
// coordinates of gtx, normally the whole window), kept inside the window.
// Windows call it once per frame, after everything else.
func (th *Theme) LayoutTooltip(gtx layout.Context, ptr image.Point) {
	t := &th.tips
	text := t.text
	t.text = ""
	for k := range t.since {
		if !t.seen[k] {
			delete(t.since, k)
		}
	}
	clear(t.seen)
	if text == "" {
		return
	}
	p := th.Pal
	g := gtx
	g.Constraints = layout.Constraints{Max: image.Pt(min(gtx.Dp(360), gtx.Constraints.Max.X), gtx.Constraints.Max.Y)}
	call, d := record(g, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 3, Bottom: 3, Left: 7, Right: 7}.Layout(gtx,
			Text{S: text, Size: Em(0.85), Color: p.TooltipFg}.Widget(th))
	})
	win := gtx.Constraints.Max
	// Below right of the pointer; above it near the bottom; inside the window.
	pos := ptr.Add(image.Pt(gtx.Dp(10), gtx.Dp(20)))
	if pos.Y+d.Size.Y > win.Y {
		pos.Y = ptr.Y - d.Size.Y - gtx.Dp(8)
	}
	pos.X = max(0, min(pos.X, win.X-d.Size.X-gtx.Dp(2)))
	pos.Y = max(0, pos.Y)
	m := op.Record(gtx.Ops)
	o := op.Offset(pos).Push(gtx.Ops)
	box(gtx, d.Size, gtx.Dp(unit.Dp(4)), p.TooltipBg, p.BorderColor)
	call.Add(gtx.Ops)
	o.Pop()
	op.Defer(gtx.Ops, m.Stop())
}
