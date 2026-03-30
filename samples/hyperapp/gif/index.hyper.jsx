const TENOR_API_KEY = "AIzaSyAyimkuYQYF_FXVALexPuGQctUWRURdCYQ"

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

      fetch(`https://tenor.googleapis.com/v2/search?q=${text}&key=${TENOR_API_KEY}&limit=1&media_filter=gif`)
        .then(data => data.json())
        .then(({ results }) => {
          actions.toggleFetching()
          results[0] && actions.setURL(results[0].media_formats.gif.url)
        })
    },
    setURL: (state, actions, url) => ({ url }),
    toggleFetching: state => ({ isFetching: !state.isFetching })
  }
})

