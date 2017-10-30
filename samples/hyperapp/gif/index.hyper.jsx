const GIPHY_API_KEY = "dc6zaTOxFJmzC"

app({
  state: {
    url: "",
    isFetching: false
  },
  view: (state, actions) =>
    <main>
      <input
        type="text"
        placeholder="Type here..."
        onkeyup={actions.getURL}
        autofocus
      />
      <div class="container">
        <img
          src={state.url}
          style={{
            display: state.isFetching || state.url === "" ? "none" : "block"
          }}
        />
      </div>
    </main>,
  actions: {
    getURL: (state, actions, { target }) => {
      const text = target.value

      if (state.isFetching || text === "") {
        return { url: "" }
      }

      actions.toggleFetching()

      fetch(`//api.giphy.com/v1/gifs/search?q=${text}&api_key=${GIPHY_API_KEY}`)
        .then(data => data.json())
        .then(({ data }) => {
          actions.toggleFetching()
          data[0] && actions.setURL(data[0].images.original.url)
        })
    },
    setURL: (state, actions, url) => ({ url }),
    toggleFetching: state => ({ isFetching: !state.isFetching })
  }
})

