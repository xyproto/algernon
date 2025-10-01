package huldra

func IsHTML(data []byte) bool {
	return HasHTMLTag(data, 200)
}
