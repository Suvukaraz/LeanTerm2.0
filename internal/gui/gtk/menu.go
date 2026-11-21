package gtk

import (
	"image"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
)

// MenuItem is an entry of a context menu.
type MenuItem struct {
	ID    string
	Label string
	// Icon is a Nerd Fonts icon (0 = none).
	Icon      rune
	Shortcut  string
	Disabled  bool
	Separator bool
	// Checked shows a check mark in the icon column.
	Checked bool
	Submenu []MenuItem
}

// Menu is a popup menu opened at a point; Activated reports the chosen item.
//
// The menu and its submenus always stay inside the window: a submenu opens to
// the right of its row, or to the left when there is no room, and a menu
// higher than the window scrolls (wheel, or the keyboard moving the focus).
type Menu struct {
	Items []MenuItem

	pos       image.Point
	open      bool
	clicks    []widget.Clickable
	sub       *Menu
	subIndex  int
	focus     int
	activated string
	dismiss   int
	tag       int
	grab      bool
	hovered   int // 1 + the row under the pointer (0: none), to follow its changes
	// opening: Open was called in this frame. The press that opened the menu
	// (a right click elsewhere while it was open) must not close it again.
	opening bool
	// dismissed: the menu was closed by a press outside (Dismissed).
	dismissed bool

	scroll  int   // pixels scrolled (menus higher than the window)
	maxScrl int   // the largest scroll
	rowTop  []int // the rows' tops and heights in the menu's content
	rowH    []int
	ensure  bool        // scroll the focused row into view
	up      bool        // OpenUp
	at, box image.Point // where the menu was drawn last, and its size
}

// Open shows the menu with its top left corner at pos (in the coordinates
// of the place where Layout is called).
func (m *Menu) Open(pos image.Point) {
	m.pos, m.open, m.focus, m.sub = pos, true, -1, nil
	m.scroll, m.opening, m.dismissed, m.up = 0, true, false, false
	m.subIndex = -1
	m.hovered = 0
	m.grab = true
}

// OpenUp shows the menu above pos (its bottom left corner there) when it
// fits, else below it: for menus of a status bar at the window's bottom.
func (m *Menu) OpenUp(pos image.Point) {
	m.Open(pos)
	m.up = true
}

// Close hides the menu.
func (m *Menu) Close() {
	m.open, m.sub = false, nil
	m.subIndex = -1
	m.focus = -1
	m.hovered = 0
}

// Opened reports whether the menu is shown.
func (m *Menu) Opened() bool { return m.open }

// Dismissed reports (once) that the last Close came from a press outside
// the menu. A menu bar uses it so that pressing the title of the open menu
// closes it instead of opening it again.
func (m *Menu) Dismissed() bool {
	d := m.dismissed
	m.dismissed = false
	return d
}

// Activated returns the ID of the item chosen since the last call.
func (m *Menu) Activated() (string, bool) {
	a := m.activated
	m.activated = ""
	if a == "" && m.sub != nil {
		return m.sub.Activated()
	}
	return a, a != ""
}

func (m *Menu) selectable(i int) bool {
	return i >= 0 && i < len(m.Items) && !m.Items[i].Separator && !m.Items[i].Disabled
}

func (m *Menu) activate(i int) {
	it := m.Items[i]
	if len(it.Submenu) > 0 {
		m.openSub(i)
		return
	}
	m.activated = it.ID
	m.open = false
}

func (m *Menu) openSub(i int) {
	if m.sub != nil && m.subIndex == i {
		return
	}
	m.sub = &Menu{Items: m.Items[i].Submenu, focus: -1, open: true, subIndex: -1}
	m.subIndex = i
}

// Layout handles input and draws the open menu on top of everything (it
// must be called where the menu's position is meant; the constraints are
// the area the menu must stay in, normally the window).
func (m *Menu) Layout(gtx layout.Context, th *Theme) layout.Dimensions {
	if !m.open {
		return layout.Dimensions{}
	}
	m.update(gtx)
	m.opening = false
	if !m.open {
		// Whoever reads Activated or Opened does so in the next frame.
		gtx.Execute(op.InvalidateCmd{})
		return layout.Dimensions{}
	}
	rec := op.Record(gtx.Ops)
	// Clicks outside the menu close it and still reach what is below (the
	// GUI closes on pointerdown without stopping the event).
	pass := pointer.PassOp{}.Push(gtx.Ops)
	big := clip.Rect{Min: image.Pt(-1<<20, -1<<20), Max: image.Pt(1<<20, 1<<20)}.Push(gtx.Ops)
	event.Op(gtx.Ops, &m.dismiss)
	big.Pop()
	pass.Pop()
	bounds := gtx.Constraints.Max
	if bounds.X <= 0 || bounds.Y <= 0 {
		bounds = image.Pt(1<<20, 1<<20)
	}
	m.draw(gtx, th, image.Rectangle{Min: m.pos, Max: m.pos}, bounds, true)
	op.Defer(gtx.Ops, rec.Stop())
	return layout.Dimensions{}
}

func (m *Menu) update(gtx layout.Context) {
	if len(m.clicks) != len(m.Items) {
		m.clicks = make([]widget.Clickable, len(m.Items))
	}
	m.scrollEvents(gtx)

	hoveredRow := -1
	for i := range m.clicks {
		if m.clicks[i].Clicked(gtx) && m.selectable(i) {
			m.activate(i)
			gtx.Execute(key.FocusCmd{Tag: &m.tag})
			return
		}
		if m.clicks[i].Hovered() && m.selectable(i) {
			hoveredRow = i
		}
	}

	if hoveredRow >= 0 {
		if m.focus != hoveredRow {
			m.focus = hoveredRow
			m.hovered = hoveredRow + 1
			if len(m.Items[hoveredRow].Submenu) > 0 {
				m.openSub(hoveredRow)
			} else {
				m.sub = nil
				m.subIndex = -1
			}
			gtx.Execute(op.InvalidateCmd{})
		}
	} else if m.sub != nil && m.sub.Opened() {
		// Mouse might be over the submenu or transitioning: keep parent item focused.
		m.focus = m.subIndex
	} else if m.hovered != 0 {
		m.hovered = 0
		m.focus = -1
		gtx.Execute(op.InvalidateCmd{})
	}

	if !m.open {
		return
	}
	if m.sub != nil {
		m.sub.update(gtx)
		if a, ok := m.sub.Activated(); ok {
			m.activated, m.open = a, false
			return
		}
	}
	// An open menu takes the keyboard (else the terminal below would).
	if m.grab {
		m.grab = false
		gtx.Execute(key.FocusCmd{Tag: &m.tag})
	}
	for {
		ev, ok := gtx.Event(
			pointer.Filter{Target: &m.dismiss, Kinds: pointer.Press},
			key.FocusFilter{Target: &m.tag},
			key.Filter{Focus: &m.tag, Name: key.NameEscape},
			key.Filter{Focus: &m.tag, Name: key.NameUpArrow},
			key.Filter{Focus: &m.tag, Name: key.NameDownArrow},
			key.Filter{Focus: &m.tag, Name: key.NameHome},
			key.Filter{Focus: &m.tag, Name: key.NameEnd},
			key.Filter{Focus: &m.tag, Name: key.NamePageUp},
			key.Filter{Focus: &m.tag, Name: key.NamePageDown},
			key.Filter{Focus: &m.tag, Name: key.NameReturn},
			key.Filter{Focus: &m.tag, Name: key.NameEnter},
			key.Filter{Focus: &m.tag, Name: key.NameSpace},
			key.Filter{Focus: &m.tag, Name: key.NameRightArrow},
			key.Filter{Focus: &m.tag, Name: key.NameLeftArrow},
		)
		if !ok {
			break
		}
		switch e := ev.(type) {
		case pointer.Event:
			// The press that opened the menu in this frame is not one outside it.
			if !m.opening && !m.inside(e.Position.Round()) {
				m.Close()
				m.dismissed = true
			}
		case key.Event:
			if e.State != key.Press {
				continue
			}
			target := m
			for target.sub != nil && target.sub.focus >= 0 {
				target = target.sub
			}
			target.key(e.Name, m)
			gtx.Execute(op.InvalidateCmd{})
		}
	}
}

// inside reports whether p (in the root menu's coordinates, the place of
// Layout) lies on the menu or one of its open submenus.
func (m *Menu) inside(p image.Point) bool {
	for s := m; s != nil; s = s.sub {
		if p.In(image.Rectangle{Min: s.at, Max: s.at.Add(s.box)}) {
			return true
		}
	}
	return false
}

// scrollEvents scrolls a menu higher than the window (wheel over it).
func (m *Menu) scrollEvents(gtx layout.Context) {
	for {
		ev, ok := gtx.Event(pointer.Filter{Target: &m.tag, Kinds: pointer.Scroll, ScrollY: pointer.ScrollRange{Min: -1 << 20, Max: 1 << 20}})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && m.maxScrl > 0 {
			m.scroll = max(0, min(m.maxScrl, m.scroll+int(e.Scroll.Y)))
			if m.sub != nil {
				m.sub = nil // its row moved
				m.subIndex = -1
			}
			gtx.Execute(op.InvalidateCmd{})
		}
	}
}

// key moves the focus within this (sub)menu; root is the outermost menu.
func (m *Menu) key(name key.Name, root *Menu) {
	move := func(start, step int) {
		for i, n := start, 0; n < len(m.Items); i, n = i+step, n+1 {
			i = (i + len(m.Items)) % len(m.Items)
			if m.selectable(i) {
				m.focus, m.ensure = i, true
				if m.sub != nil && m.subIndex != i {
					m.sub = nil
					m.subIndex = -1
				}
				return
			}
		}
	}
	switch name {
	case key.NameEscape:
		// Escape closes the innermost menu only.
		if m == root {
			root.Close()
			return
		}
		for p := root; p.sub != nil; p = p.sub {
			if p.sub == m {
				p.sub = nil
				p.subIndex = -1
				return
			}
		}
	case key.NameDownArrow, key.NameUpArrow:
		step := 1
		if name == key.NameUpArrow {
			step = -1
		}
		start := m.focus + step
		if m.focus < 0 && step < 0 {
			start = len(m.Items) - 1 // Up without a focused row: the last one
		}
		move(start, step)
	case key.NameHome, key.NamePageUp:
		move(0, 1)
	case key.NameEnd, key.NamePageDown:
		move(len(m.Items)-1, -1)
	case key.NameReturn, key.NameEnter, key.NameSpace, key.NameRightArrow:
		if m.selectable(m.focus) {
			if len(m.Items[m.focus].Submenu) > 0 {
				m.openSub(m.focus)
				m.sub.focus = -1
				m.sub.key(key.NameDownArrow, root) // its first selectable row
			} else if name != key.NameRightArrow {
				m.activate(m.focus)
				root.activated, root.open = m.activated, false
			}
		}
	case key.NameLeftArrow:
		// Back to the parent: find it from the root.
		for p := root; p.sub != nil; p = p.sub {
			if p.sub == m {
				p.sub = nil
				p.subIndex = -1
				break
			}
		}
	}
}

// draw lays the menu out next to anchor (a point for the root menu, the
// row of the parent for a submenu), inside bounds, and its open submenu
// next to its row. All positions are relative to the place of Layout.
func (m *Menu) draw(gtx layout.Context, th *Theme, anchor image.Rectangle, bounds image.Point, root bool) {
	p := th.Pal
	// A submenu opened by a key this frame has not been updated yet.
	if len(m.clicks) != len(m.Items) {
		m.clicks = make([]widget.Clickable, len(m.Items))
	}
	if len(m.rowTop) != len(m.Items) {
		m.rowTop, m.rowH = make([]int, len(m.Items)), make([]int, len(m.Items))
	}
	orig := gtx.Constraints
	// min-width 200px, max-width 420px, else as wide as the widest row.
	width := gtx.Dp(200)
	for i, it := range m.Items {
		if it.Separator {
			continue
		}
		g := gtx
		g.Constraints = layout.Constraints{Max: image.Pt(gtx.Dp(420), 1<<20)}
		_, d := record(g, func(gtx layout.Context) layout.Dimensions { return m.rowContent(gtx, th, i, it, false, true) })
		width = max(width, min(d.Size.X, gtx.Dp(420)))
	}
	width = min(width, bounds.X)
	gtx.Constraints = layout.Exact(image.Pt(width, 0))
	gtx.Constraints.Max.Y = 1 << 20
	call, d := record(gtx, func(gtx layout.Context) layout.Dimensions {
		var rows []layout.FlexChild
		y := gtx.Dp(4)
		for i, it := range m.Items {
			i, it := i, it
			rows = append(rows, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
				var dims layout.Dimensions
				if it.Separator {
					dims = layout.Inset{Top: 4, Bottom: 4, Left: 6, Right: 6}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return HLine(gtx, p.BorderColor)
					})
				} else {
					dims = m.row(gtx, th, i, it)
				}
				m.rowTop[i], m.rowH[i] = y, dims.Size.Y
				y += dims.Size.Y
				return dims
			}))
		}
		return layout.Inset{Top: 4, Bottom: 4}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, rows...)
		})
	})
	gtx.Constraints = orig

	// The visible part: at most the window's height.
	vis := image.Pt(d.Size.X, min(d.Size.Y, bounds.Y))
	m.maxScrl = d.Size.Y - vis.Y
	if m.ensure && m.focus >= 0 && m.focus < len(m.rowTop) {
		top, bot := m.rowTop[m.focus], m.rowTop[m.focus]+m.rowH[m.focus]
		if top < m.scroll {
			m.scroll = top - gtx.Dp(4)
		} else if bot > m.scroll+vis.Y {
			m.scroll = bot - vis.Y + gtx.Dp(4)
		}
	}
	m.ensure = false
	m.scroll = max(0, min(m.maxScrl, m.scroll))

	// The position: the root menu at its point, a submenu right of its row
	// (or left of it without room there), both kept inside bounds.
	pos := anchor.Min
	if root && m.up && pos.Y-vis.Y >= 0 {
		pos.Y -= vis.Y
	}
	if !root {
		pos = image.Pt(anchor.Max.X-gtx.Dp(2), anchor.Min.Y-gtx.Dp(4))
		if pos.X+vis.X > bounds.X {
			pos.X = anchor.Min.X - vis.X + gtx.Dp(2)
		}
	}
	pos.X = max(0, min(pos.X, bounds.X-vis.X))
	pos.Y = max(0, min(pos.Y, bounds.Y-vis.Y))
	m.at, m.box = pos, vis

	o := offset(gtx, pos.X, pos.Y)
	// box-shadow 0 6px 18px, border 1px, radius 5px.
	for i := 1; i <= 4; i++ {
		s := gtx.Dp(unit.Dp(i * 3))
		r := image.Rectangle{Min: image.Pt(-s/2, -s/2+gtx.Dp(6)), Max: vis.Add(image.Pt(s/2, s/2+gtx.Dp(6)))}
		paint.FillShape(gtx.Ops, faded(p.Shadow, 0.15), clip.UniformRRect(r, gtx.Dp(5)+s/2).Op(gtx.Ops))
	}
	box(gtx, vis, gtx.Dp(5), p.MenuBg, p.BorderColor)
	// Clicks between the rows stay in the menu; the rows lie inside its
	// area, which also takes the wheel.
	area := clip.UniformRRect(image.Rectangle{Max: vis}, gtx.Dp(5)).Push(gtx.Ops)
	event.Op(gtx.Ops, &m.tag)
	so := offset(gtx, 0, -m.scroll)
	call.Add(gtx.Ops)
	so.Pop()
	area.Pop()
	if m.maxScrl > 0 {
		// A thin indicator where the visible part is.
		h := max(gtx.Dp(16), vis.Y*vis.Y/d.Size.Y)
		y := (vis.Y - h) * m.scroll / m.maxScrl
		FillRect(gtx, image.Rect(vis.X-gtx.Dp(4), y+gtx.Dp(2), vis.X-gtx.Dp(2), y+h-gtx.Dp(2)), faded(p.FgColor, 0.35))
	}
	o.Pop()
	if m.sub != nil && m.subIndex >= 0 && m.subIndex < len(m.rowTop) {
		row := image.Rect(pos.X, pos.Y+m.rowTop[m.subIndex]-m.scroll, pos.X+vis.X, pos.Y+m.rowTop[m.subIndex]-m.scroll+m.rowH[m.subIndex])
		m.sub.draw(gtx, th, row, bounds, false)
	}
}

// row draws an item: padding 4px 10px, gap 8px, a 16px icon column, the
// label, the shortcut (0.85em, muted) and the submenu arrow.
func (m *Menu) row(gtx layout.Context, th *Theme, i int, it MenuItem) layout.Dimensions {
	c := &m.clicks[i]
	hot := m.selectable(i) && m.focus == i
	return c.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		pointer.CursorDefault.Add(gtx.Ops)
		return m.rowContent(gtx, th, i, it, hot, false)
	})
}

// rowContent draws a row; measure lays it out at its natural width.
func (m *Menu) rowContent(gtx layout.Context, th *Theme, i int, it MenuItem, hot, measure bool) layout.Dimensions {
	p := th.Pal
	fg, muted := p.FgColor, p.MutedFg
	if hot {
		fg, muted = p.SelectedFg, faded(p.SelectedFg, 0.8)
	}
	if it.Disabled {
		fg, muted = dim(fg, p.MenuBg, 0.45), dim(muted, p.MenuBg, 0.45)
	}
	call, d := record(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: 4, Bottom: 4, Left: 10, Right: 10}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			if !measure {
				gtx.Constraints.Min.X = gtx.Constraints.Max.X
			}
			label := layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return Text{S: it.Label, Size: Em(0.95), Color: fg, MaxLines: 1}.Layout(gtx, th)
			})
			if !measure {
				label = layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					d := Text{S: it.Label, Size: Em(0.95), Color: fg, MaxLines: 1}.Layout(gtx, th)
					d.Size.X = gtx.Constraints.Max.X
					return d
				})
			}
			return layout.Flex{Alignment: layout.Middle}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					switch {
					case it.Checked:
						return th.IconBox(gtx, '\uf00c', 16, fg) // nf-fa-check
					case it.Icon != 0:
						return th.IconBox(gtx, it.Icon, 16, fg)
					}
					return layout.Dimensions{Size: image.Pt(gtx.Dp(16), gtx.Dp(16))}
				}),
				layout.Rigid(layout.Spacer{Width: 8}.Layout),
				label,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					var meta []layout.FlexChild
					if it.Shortcut != "" {
						meta = append(meta, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return Text{S: it.Shortcut, Size: Em(0.95 * 0.85), Color: muted, MaxLines: 1}.Layout(gtx, th)
						}))
					}
					if len(it.Submenu) > 0 {
						if len(meta) > 0 {
							meta = append(meta, layout.Rigid(layout.Spacer{Width: 6}.Layout))
						}
						meta = append(meta, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return th.Icon(gtx, '\uf0da', Em(0.95*0.75), fg) // nf-fa-caret_right
						}))
					}
					if len(meta) == 0 {
						return layout.Dimensions{}
					}
					return layout.Inset{Left: 12}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Alignment: layout.Middle}.Layout(gtx, meta...)
					})
				}),
			)
		})
	})
	if hot {
		FillRect(gtx, image.Rectangle{Max: d.Size}, p.SelectedBg)
	}
	call.Add(gtx.Ops)
	return d
}
