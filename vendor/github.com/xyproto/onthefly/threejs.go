package onthefly

import (
	"fmt"
	"log"
)

// For generating IDs
var (
	geometryCounter = 0
	materialCounter = 0
	meshCounter     = 0
)

// Unique prefixes when generating IDs
const (
	geometryPrefix = "g"
	materialPrefix = "ma"
	meshPrefix     = "m"
)

type (
	// For Three.JS elements, like a mesh or material
	Element struct {
		ID string // name of the variable
		JS string // javascript code for creating the element
	}
	// The Three.JS render function, where head and tail are standard
	RenderFunc struct {
		head, mid, tail string
	}
	// Different types of elements
	Geometry Element
	Material Element
	Mesh     Element
)

// Create a HTML5 page that links with Three.JS and sets up a scene
func NewThreeJS(titleText string) (*Page, *Tag) {
	page := NewHTML5Page(titleText)

	// Style the page for showing a fullscreen canvas
	page.FullCanvas()

	// Link to Three.JS
	page.LinkToJSInBody("http://threejs.org/build/three.min.js")

	// Add a scene
	script, _ := page.AddScriptToBody("var scene = new THREE.Scene();")

	// Return the script tag that can be used for adding additional
	// javascript/Three.JS code
	return page, script
}

// Create a HTML5 page that includes the given JavaScript code at the end of
// the <body> tag, before </body>. The given JS code must be the contents of
// the http://threejs.org/build/three.min.js script for this to work.
func NewThreeJSWithGiven(titleText, js string) (*Page, *Tag) {
	page := NewHTML5Page(titleText)

	// Style the page for showing a fullscreen canvas
	page.FullCanvas()

	// Add the given JavaScript to the body
	page.AddScriptToBody(js)

	// Add a scene
	script, _ := page.AddScriptToBody("var scene = new THREE.Scene();")

	// Return the script tag that can be used for adding additional
	// javascript/Three.JS code
	return page, script
}

// Add a camera with default settings
func (three *Tag) AddCamera() {
	// TODO create an AddCustomCamera function
	three.AddContent("var camera = new THREE.PerspectiveCamera(75, window.innerWidth/window.innerHeight, 0.1, 1000);")
}

// Add a WebGL renderer with default settings
func (three *Tag) AddRenderer() {
	three.AddContent("var renderer = new THREE.WebGLRenderer();")
	three.AddContent("renderer.setSize(window.innerWidth, window.innerHeight);")
	three.AddContent("document.body.appendChild(renderer.domElement);")
}

// Add a mesh to the current scene
func (three *Tag) AddToScene(mesh *Mesh) {
	three.AddContent(mesh.JS)
	three.AddContent("scene.add(" + mesh.ID + ");")
}

// Create a new mesh, given geometry and material.
// The geometry and material will be instanciated together with the mesh.
func NewMesh(geometry *Geometry, material *Material) *Mesh {
	id := fmt.Sprintf("%s%d", meshPrefix, meshCounter)
	meshCounter++
	js := geometry.JS + material.JS
	js += "var " + id + " = new THREE.Mesh(" + geometry.ID + ", " + material.ID + ");"
	return &Mesh{id, js}
}

// Set the camera position. Axis must be "x", "y", or "z".
func (three *Tag) CameraPos(axis string, value int) {
	if (axis != "x") && (axis != "y") && (axis != "z") {
		log.Fatalln("camera axis must be x, y or z")
	}
	three.AddContent(fmt.Sprintf("camera.position.%s = %d;", axis, value))
}

// Very simple type of material
func NewMaterial(color string) *Material {
	id := fmt.Sprintf("%s%d", materialPrefix, materialCounter)
	materialCounter++
	js := "var " + id + " = new THREE.MeshBasicMaterial({color: " + color + "});"
	return &Material{id, js}
}

// Create a material which reflects the normals of the geometry
func NewNormalMaterial() *Material {
	id := fmt.Sprintf("%s%d", materialPrefix, materialCounter)
	materialCounter++
	js := "var " + id + " = new THREE.MeshNormalMaterial();"
	return &Material{id, js}
}

// Create geometry for a box
func NewBoxGeometry(w, h, d int) *Geometry {
	id := fmt.Sprintf("%s%d", geometryPrefix, geometryCounter)
	geometryCounter++
	js := fmt.Sprintf("var %s = new THREE.BoxGeometry(%d, %d, %d);", id, w, h, d)
	return &Geometry{id, js}
}

// Add a test cube to the scene
func (three *Tag) AddTestCube() *Mesh {
	// TODO Create functions for adding geometry, material and creating meshes
	//material := NewMaterial(color)
	material := NewNormalMaterial()
	geometry := NewBoxGeometry(1, 1, 1)
	cube := NewMesh(geometry, material)
	three.AddToScene(cube)
	return cube
}

// Create a new render function, which is called at every animation frame
func NewRenderFunction() *RenderFunc {
	head := "var render = function() { requestAnimationFrame(render);"
	tail := "renderer.render(scene, camera); };"
	return &RenderFunc{head, "", tail}
}

// Add javascript code to the body of a render function
func (r *RenderFunc) AddJS(s string) {
	r.mid += s
}

// Add a render function.
// If call is true, the render function is called at the end of the script.
func (three *Tag) AddRenderFunction(r *RenderFunc, call bool) {
	three.AddContent(r.head + r.mid + r.tail)
	if call {
		three.AddContent("render();")
	}
}
