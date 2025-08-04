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
	cameraCounter   = 0
	lightCounter    = 0
	rendererCounter = 0
)

// Unique prefixes when generating IDs
const (
	geometryPrefix = "g"
	materialPrefix = "ma"
	meshPrefix     = "m"
	cameraPrefix   = "cam"
	lightPrefix    = "light"
	rendererPrefix = "renderer"
)

type (
	// Element represents Three.JS elements, like a mesh or material
	Element struct {
		ID string // name of the variable
		JS string // javascript code for creating the element
	}
	// RenderFunc represents the Three.JS render function, where head and tail are standard
	RenderFunc struct {
		head, mid, tail string
	}
	// Geometry represents a Three.JS geometry
	Geometry Element
	// Material represents a Three.JS material
	Material Element
	// Mesh represents a Three.JS mesh
	Mesh Element
	// Camera represents a Three.JS camera
	Camera Element
	// Light represents a Three.JS light
	Light Element
	// Renderer represents a Three.JS renderer
	Renderer Element
)

// NewThreeJS creates a HTML5 page that links with Three.JS and sets up a scene
func NewThreeJS(args ...string) (*Page, *Tag) {
	title := "Untitled"
	if len(args) > 0 {
		title = args[0]
	}

	page := NewHTML5Page(title)

	// Style the page for showing a fullscreen canvas
	page.FullCanvas()

	if len(args) > 1 {
		threeJSURL := args[1]
		page.LinkToJSInBody(threeJSURL)
	} else {
		// Link to Three.JS
		page.LinkToJSInBody("https://threejs.org/build/three.min.js")
	}

	// Add a scene
	script, _ := page.AddScriptToBody("var scene = new THREE.Scene();")

	// Return the script tag that can be used for adding additional
	// javascript/Three.JS code
	return page, script
}

// NewThreeJSWithGiven creates a HTML5 page that includes the given JavaScript code at the end of
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

// AddCamera adds a camera with default settings
func (three *Tag) AddCamera() {
	three.AddContent("var camera = new THREE.PerspectiveCamera(75, window.innerWidth/window.innerHeight, 0.1, 1000);")
}

// NewPerspectiveCamera creates a new perspective camera with custom parameters
func NewPerspectiveCamera(fov, aspect, near, far float64) *Camera {
	id := fmt.Sprintf("%s%d", cameraPrefix, cameraCounter)
	cameraCounter++
	js := fmt.Sprintf("var %s = new THREE.PerspectiveCamera(%g, %g, %g, %g);", id, fov, aspect, near, far)
	return &Camera{id, js}
}

// NewOrthographicCamera creates a new orthographic camera
func NewOrthographicCamera(left, right, top, bottom, near, far float64) *Camera {
	id := fmt.Sprintf("%s%d", cameraPrefix, cameraCounter)
	cameraCounter++
	js := fmt.Sprintf("var %s = new THREE.OrthographicCamera(%g, %g, %g, %g, %g, %g);", id, left, right, top, bottom, near, far)
	return &Camera{id, js}
}

// AddRenderer adds a WebGL renderer with default settings
func (three *Tag) AddRenderer() {
	three.AddContent("var renderer = new THREE.WebGLRenderer();")
	three.AddContent("renderer.setSize(window.innerWidth, window.innerHeight);")
	three.AddContent("document.body.appendChild(renderer.domElement);")
}

// NewWebGLRenderer creates a new WebGL renderer with custom options
func NewWebGLRenderer(antialias bool) *Renderer {
	id := fmt.Sprintf("%s%d", rendererPrefix, rendererCounter)
	rendererCounter++
	js := fmt.Sprintf("var %s = new THREE.WebGLRenderer({antialias: %t});", id, antialias)
	js += fmt.Sprintf("%s.setSize(window.innerWidth, window.innerHeight);", id)
	js += fmt.Sprintf("document.body.appendChild(%s.domElement);", id)
	return &Renderer{id, js}
}

// SetShadowMap enables shadow mapping for a renderer
func (r *Renderer) SetShadowMap(enabled bool) {
	r.JS += fmt.Sprintf("%s.shadowMap.enabled = %t;", r.ID, enabled)
	r.JS += fmt.Sprintf("%s.shadowMap.type = THREE.PCFSoftShadowMap;", r.ID)
}

// AddToScene adds a mesh to the current scene
func (three *Tag) AddToScene(mesh *Mesh) {
	three.AddContent(mesh.JS)
	three.AddContent("scene.add(" + mesh.ID + ");")
}

// AddElementToScene adds any Three.js element to the current scene
func (three *Tag) AddElementToScene(element *Element) {
	three.AddContent(element.JS)
	three.AddContent("scene.add(" + element.ID + ");")
}

// AddLightToScene adds a light to the current scene
func (three *Tag) AddLightToScene(light *Light) {
	three.AddContent(light.JS)
	three.AddContent("scene.add(" + light.ID + ");")
}

// AddCameraToScene adds a camera element to the scene (for helper visualization)
func (three *Tag) AddCameraToScene(camera *Camera) {
	three.AddContent(camera.JS)
}

// NewMesh creates a new mesh, given geometry and material.
// The geometry and material will be instanciated together with the mesh.
func NewMesh(geometry *Geometry, material *Material) *Mesh {
	id := fmt.Sprintf("%s%d", meshPrefix, meshCounter)
	meshCounter++
	js := geometry.JS + material.JS
	js += "var " + id + " = new THREE.Mesh(" + geometry.ID + ", " + material.ID + ");"
	return &Mesh{id, js}
}

// CameraPos sets the camera position. Axis must be "x", "y", or "z".
func (three *Tag) CameraPos(axis string, value float64) {
	if (axis != "x") && (axis != "y") && (axis != "z") {
		log.Fatalln("camera axis must be x, y or z")
	}
	three.AddContent(fmt.Sprintf("camera.position.%s = %g;", axis, value))
}

// SetPosition sets the position of any Three.js object
func (e *Element) SetPosition(x, y, z float64) {
	e.JS += fmt.Sprintf("%s.position.set(%g, %g, %g);", e.ID, x, y, z)
}

// SetRotation sets the rotation of any Three.js object
func (e *Element) SetRotation(x, y, z float64) {
	e.JS += fmt.Sprintf("%s.rotation.set(%g, %g, %g);", e.ID, x, y, z)
}

// SetScale sets the scale of any Three.js object
func (e *Element) SetScale(x, y, z float64) {
	e.JS += fmt.Sprintf("%s.scale.set(%g, %g, %g);", e.ID, x, y, z)
}

// NewMaterial creates a very simple type of material
func NewMaterial(color string) *Material {
	id := fmt.Sprintf("%s%d", materialPrefix, materialCounter)
	materialCounter++
	js := "var " + id + " = new THREE.MeshBasicMaterial({color: " + color + "});"
	return &Material{id, js}
}

// NewNormalMaterial creates a material which reflects the normals of the geometry
func NewNormalMaterial() *Material {
	id := fmt.Sprintf("%s%d", materialPrefix, materialCounter)
	materialCounter++
	js := "var " + id + " = new THREE.MeshNormalMaterial();"
	return &Material{id, js}
}

// NewLambertMaterial creates a Lambert material (responds to lighting)
func NewLambertMaterial(color string) *Material {
	id := fmt.Sprintf("%s%d", materialPrefix, materialCounter)
	materialCounter++
	js := fmt.Sprintf("var %s = new THREE.MeshLambertMaterial({color: %s});", id, color)
	return &Material{id, js}
}

// NewPhongMaterial creates a Phong material (supports shiny surfaces)
func NewPhongMaterial(color string) *Material {
	id := fmt.Sprintf("%s%d", materialPrefix, materialCounter)
	materialCounter++
	js := fmt.Sprintf("var %s = new THREE.MeshPhongMaterial({color: %s});", id, color)
	return &Material{id, js}
}

// NewStandardMaterial creates a standard material (physically based)
func NewStandardMaterial(color string) *Material {
	id := fmt.Sprintf("%s%d", materialPrefix, materialCounter)
	materialCounter++
	js := fmt.Sprintf("var %s = new THREE.MeshStandardMaterial({color: %s});", id, color)
	return &Material{id, js}
}

// NewBoxGeometry creates geometry for a box
func NewBoxGeometry(w, h, d float64) *Geometry {
	id := fmt.Sprintf("%s%d", geometryPrefix, geometryCounter)
	geometryCounter++
	js := fmt.Sprintf("var %s = new THREE.BoxGeometry(%g, %g, %g);", id, w, h, d)
	return &Geometry{id, js}
}

// NewSphereGeometry creates geometry for a sphere
func NewSphereGeometry(radius float64, widthSegments, heightSegments int) *Geometry {
	id := fmt.Sprintf("%s%d", geometryPrefix, geometryCounter)
	geometryCounter++
	js := fmt.Sprintf("var %s = new THREE.SphereGeometry(%g, %d, %d);", id, radius, widthSegments, heightSegments)
	return &Geometry{id, js}
}

// NewPlaneGeometry creates geometry for a plane
func NewPlaneGeometry(width, height float64) *Geometry {
	id := fmt.Sprintf("%s%d", geometryPrefix, geometryCounter)
	geometryCounter++
	js := fmt.Sprintf("var %s = new THREE.PlaneGeometry(%g, %g);", id, width, height)
	return &Geometry{id, js}
}

// NewCylinderGeometry creates geometry for a cylinder
func NewCylinderGeometry(radiusTop, radiusBottom, height float64, radialSegments int) *Geometry {
	id := fmt.Sprintf("%s%d", geometryPrefix, geometryCounter)
	geometryCounter++
	js := fmt.Sprintf("var %s = new THREE.CylinderGeometry(%g, %g, %g, %d);", id, radiusTop, radiusBottom, height, radialSegments)
	return &Geometry{id, js}
}

// AddTestCube adds a test cube to the scene
func (three *Tag) AddTestCube() *Mesh {
	material := NewNormalMaterial()
	geometry := NewBoxGeometry(1, 1, 1)
	cube := NewMesh(geometry, material)
	three.AddToScene(cube)
	return cube
}

// NewAmbientLight creates an ambient light that illuminates all objects equally
func NewAmbientLight(color string, intensity float64) *Light {
	id := fmt.Sprintf("%s%d", lightPrefix, lightCounter)
	lightCounter++
	js := fmt.Sprintf("var %s = new THREE.AmbientLight(%s, %g);", id, color, intensity)
	return &Light{id, js}
}

// NewDirectionalLight creates a directional light (like sunlight)
func NewDirectionalLight(color string, intensity float64) *Light {
	id := fmt.Sprintf("%s%d", lightPrefix, lightCounter)
	lightCounter++
	js := fmt.Sprintf("var %s = new THREE.DirectionalLight(%s, %g);", id, color, intensity)
	return &Light{id, js}
}

// NewPointLight creates a point light (like a light bulb)
func NewPointLight(color string, intensity, distance float64) *Light {
	id := fmt.Sprintf("%s%d", lightPrefix, lightCounter)
	lightCounter++
	js := fmt.Sprintf("var %s = new THREE.PointLight(%s, %g, %g);", id, color, intensity, distance)
	return &Light{id, js}
}

// NewRenderFunction creates a new render function, which is called at every animation frame
func NewRenderFunction() *RenderFunc {
	head := "var render = function() { requestAnimationFrame(render);"
	tail := "renderer.render(scene, camera); };"
	return &RenderFunc{head, "", tail}
}

// AddJS adds javascript code to the body of a render function
func (r *RenderFunc) AddJS(s string) {
	r.mid += s
}

// AddRenderFunction adds a render function.
// If call is true, the render function is called at the end of the script.
func (three *Tag) AddRenderFunction(r *RenderFunc, call bool) {
	three.AddContent(r.head + r.mid + r.tail)
	if call {
		three.AddContent("render();")
	}
}

// AddWindowResizeHandler adds a window resize handler to keep the renderer responsive
func (three *Tag) AddWindowResizeHandler(cameraID, rendererID string) {
	resizeJS := fmt.Sprintf(`
window.addEventListener('resize', function() {
	%s.aspect = window.innerWidth / window.innerHeight;
	%s.updateProjectionMatrix();
	%s.setSize(window.innerWidth, window.innerHeight);
});`, cameraID, cameraID, rendererID)
	three.AddContent(resizeJS)
}
