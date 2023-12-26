class TodoList extends React.Component {
  render() {
    const createItem = (itemText, index) => <li key={index + itemText}>{itemText}</li>;
    return <ul>{this.props.items.map(createItem)}</ul>;
  }
}

class TodoApp extends React.Component {
  constructor(props) {
    super(props);
    this.state = { items: [], text: '' };
  }

  onChange = (e) => {
    this.setState({ text: e.target.value });
  }

  handleSubmit = (e) => {
    e.preventDefault();
    const nextItems = this.state.items.concat([this.state.text]);
    const nextText = '';
    this.setState({ items: nextItems, text: nextText });
  }

  render() {
    return (
      <div>
        <h3>TODO</h3>
        <TodoList items={this.state.items} />
        <form onSubmit={this.handleSubmit}>
          <input onChange={this.onChange} value={this.state.text} />
          <button>{'Add #' + (this.state.items.length + 1)}</button>
        </form>
      </div>
    );
  }
}

ReactDOM.render(<TodoApp />, document.getElementById('content'));
