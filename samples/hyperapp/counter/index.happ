app({
  state: 0,
  view: (state, actions) => (
    <main>
      <h1>{state}</h1>
      <button onclick={actions.add}>+</button>
      <button onclick={actions.sub} disabled={state <= 0}>-</button>
    </main>
  ),
  actions: {
    add: state => state + 1,
    sub: state => state - 1
  }
})
