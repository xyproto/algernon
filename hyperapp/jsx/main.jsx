const { h, app } = hyperapp

app({
  state: "Hi.",
  view: state => <h1>{state}</h1>
})
