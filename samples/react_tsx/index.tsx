// React: 19
import { useState, useEffect } from "react";
import ReactDOM from "react-dom/client";

interface Comment {
  author: string;
  text: string;
}

function App() {
  const [comments, setComments] = useState<Comment[]>([]);
  const [author, setAuthor] = useState("");
  const [text, setText] = useState("");

  const load = () =>
    fetch("comments.lua").then((r) => r.json()).then(setComments);

  useEffect(() => {
    load();
    const id = setInterval(load, 2000);
    return () => clearInterval(id);
  }, []);

  function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!author.trim() || !text.trim()) return;
    postForm("comments.lua", { author, text }).then(load);
    setAuthor("");
    setText("");
  }

  return (
    <div className="card">
      <h1>Comments</h1>
      {comments.map((c, i) => (
        <p key={i}>
          <strong>{c.author}:</strong> {c.text}
        </p>
      ))}
      <form onSubmit={submit}>
        <input
          value={author}
          onChange={(e) => setAuthor(e.target.value)}
          placeholder="Your name"
        />
        <input
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Say something..."
        />
        <button type="submit">Post</button>
      </form>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
