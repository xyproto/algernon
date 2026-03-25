function TodoList(props) {
  return (
    <ul>
      {props.items.map((itemText, index) => (
        <li key={index + itemText}>{itemText}</li>
      ))}
    </ul>
  );
}

function TodoApp() {
  const [items, setItems] = React.useState([]);
  const [text, setText] = React.useState('');

  function handleChange(e) {
    setText(e.target.value);
  }

  function handleSubmit(e) {
    e.preventDefault();
    setItems(prev => [...prev, text]);
    setText('');
  }

  return (
    <div>
      <h3>TODO</h3>
      <TodoList items={items} />
      <form onSubmit={handleSubmit}>
        <input onChange={handleChange} value={text} />
        <button>{'Add #' + (items.length + 1)}</button>
      </form>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('content')).render(<TodoApp />);
