package onthefly

// Sample Tiny SVG drawing 1
func SampleSVG1() *Page {
	page, svg := NewTinySVG(0, 0, 30, 30)
	desc := svg.AddNewTag("desc")
	desc.AddContent("Sample SVG file 1")
	rect := svg.AddRect(10, 10, 10, 10)
	rect.Fill("green")
	svg.Pixel(10, 10, 255, 0, 0)
	svg.AlphaDot(5, 5, 0, 0, 255, 0.5)
	return page
}

// Sample Tiny SVG drawing 2
func SampleSVG2() *Page {
	w := 160
	h := 90
	stepx := 8
	stepy := 8
	page, svg := NewTinySVG(0, 0, w, h)
	desc := svg.AddNewTag("desc")
	desc.AddContent("Sample SVG file 2")
	increase := 0
	decrease := 0
	for y := stepy; y < h; y += stepy {
		for x := stepx; x < w; x += stepx {
			increase = int((float32(x) / float32(w)) * 255.0)
			decrease = 255 - increase
			svg.Dot(x, y, 255, decrease, increase)
		}
	}
	return page
}

// Sample OnTheFly-page (generates HTML5+CSS)
func SamplePage(cssurl string) *Page {
	page := NewHTML5Page("Hello")
	body, _ := page.SetMargin(3)

	h1 := body.AddNewTag("h1")
	h1.SetMargin(1)
	h1.AddContent("On")

	h1, err := page.root.GetTag("h1")
	if err == nil {
		h1.AddContent("The")
	}

	if err := page.LinkToCSS(cssurl); err == nil {
		h1.AddContent("Fly")
	} else {
		h1.AddContent("Flyyyyyyy")
	}

	page.SetColor("#202020", "#A0A0A0")
	page.SetFontFamily("sans serif")

	box, _ := page.addBox("box0", true)
	box.AddStyle("margin-top", "-2em")
	box.AddStyle("margin-bottom", "3em")

	image := body.AddImage("http://www.shoutmeloud.com/wp-content/uploads/2010/01/successful-Blogger.jpeg", "50%")
	image.AddStyle("margin-top", "2em")
	image.AddStyle("margin-left", "3em")

	return page
}

// SampleStar draws a star at the given position
func SampleStar(svg *Tag) {
	points, err := PointsFromString("350,75 379,161 469,161 397,215 423,301 350,250 277,301 303,215 231,161 321,161")
	if err != nil {
		panic(err)
	}
	polygon := svg.Polygon(points, NewColor("blue"))
	polygon.Fill("red")
}
