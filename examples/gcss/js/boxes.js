// New scene
var scene = new THREE.Scene();

// New camera
var camera = new THREE.PerspectiveCamera(75, window.innerWidth/window.innerHeight, 0.1, 1000);
camera.position.z = 5;

// New renderer
var renderer = new THREE.WebGLRenderer({alpha: true});
//var renderer = new THREE.WebGLRenderer({alpha: true, antialias: true});
renderer.setSize(window.innerWidth, window.innerHeight);
//renderer.setClearColor(0xffffff, 0);

// Add renderer to page
var div = document.getElementById("bg");
div.appendChild(renderer.domElement);

// LIGHTS

var ambient = new THREE.AmbientLight( 0x050505 );
scene.add( ambient );

directionalLight = new THREE.DirectionalLight( 0xffffff, 2 );
directionalLight.position.set( 2, 1.2, 10 ).normalize();
scene.add( directionalLight );

directionalLight = new THREE.DirectionalLight( 0xffffff, 1 );
directionalLight.position.set( -2, 1.2, -10 ).normalize();
scene.add( directionalLight );

pointLight = new THREE.PointLight( 0xffaa00, 2 );
pointLight.position.set( 2000, 1200, 10000 );
scene.add( pointLight );

var ma0 = new THREE.MeshNormalMaterial();
// torus: radius, diameter of tube, segments around radius, segments around torus
var m0 = new THREE.Mesh( new THREE.TorusGeometry( 2, 0.25, 40, 40 ), ma0);
m0.position.set(0, 0, 0);
scene.add(m0);

camera.lookAt(m0.position);

var ma1 = new THREE.MeshNormalMaterial();
// torus: radius, diameter of tube, segments around radius, segments around torus
var m1 = new THREE.Mesh( new THREE.TorusGeometry( 2, 0.25, 40, 40 ), ma1);
m1.position.set(0, 0, 0);
scene.add(m1);

var ma2 = new THREE.MeshNormalMaterial();
// torus: radius, diameter of tube, segments around radius, segments around torus
var m2 = new THREE.Mesh( new THREE.TorusGeometry( 2, 0.25, 40, 40 ), ma2);
m2.position.set(0, 0, 0);
scene.add(m2);

m1.rotation.y += 1;
m2.rotation.x += 1;

// Each frame
var render = function() {
  requestAnimationFrame(render);

  // Rotate the boxes
  m0.rotation.x += 0.0005;
  m0.rotation.y += 0.0010;
  m1.rotation.x += 0.0015;
  m1.rotation.y += 0.0020;
  m2.rotation.x += 0.0025;
  m2.rotation.y += 0.0030;

  // Render
  renderer.render(scene, camera);
};

// Start the animation
render();


