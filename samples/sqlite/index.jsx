// React: 19

function TodoItem({ todo, onToggle, onDelete }) {
  return (
    <li className={todo.done ? 'done' : ''}>
      <input type="checkbox" checked={!!todo.done} onChange={() => onToggle(todo.id)} />
      <span>{todo.text}</span>
      <button className="del" onClick={() => onDelete(todo.id)}>✕</button>
    </li>
  );
}

function TodoApp() {
  const [todos, setTodos] = React.useState([]);
  const [text, setText] = React.useState('');

  function load() {
    fetch('todo.tl').then(r => r.json()).then(setTodos);
  }

  React.useEffect(load, []);

  function handleSubmit(e) {
    e.preventDefault();
    if (!text.trim()) return;
    postForm('todo.tl', { action: 'add', text }).then(setTodos);
    setText('');
  }

  function handleToggle(id) {
    postForm('todo.tl', { action: 'toggle', id }).then(setTodos);
  }

  function handleDelete(id) {
    postForm('todo.tl', { action: 'delete', id }).then(setTodos);
  }

  return (
    <div>
      <h3>TODO</h3>
      <ul>
        {todos.map(todo => (
          <TodoItem key={todo.id} todo={todo} onToggle={handleToggle} onDelete={handleDelete} />
        ))}
      </ul>
      <form onSubmit={handleSubmit}>
        <input className="text-input" value={text} onChange={e => setText(e.target.value)} placeholder="New item…" />
        <button>{'Add #' + (todos.length + 1)}</button>
      </form>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('content')).render(<TodoApp />);
