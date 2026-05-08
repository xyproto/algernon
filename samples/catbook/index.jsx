function timeAgo(ns) {
  var sec = Math.floor((Date.now() * 1e6 - Number(ns)) / 1e9);
  if (sec < 60) return "just now";
  if (sec < 3600) return Math.floor(sec / 60) + "m ago";
  if (sec < 86400) return Math.floor(sec / 3600) + "h ago";
  return Math.floor(sec / 86400) + "d ago";
}

function Nav({ page, setPage, user, onLogout }) {
  return (
    <nav style={{display:"flex",gap:"1em",padding:"0.8em 1em",background:"#1a1a2e",color:"#fff",alignItems:"center"}}>
      <span style={{fontSize:"1.3em",cursor:"pointer"}} onClick={() => setPage("feed")}>🐱 Catbook</span>
      <a href="#" onClick={() => setPage("feed")} style={{color: page==="feed"?"#ffd700":"#aaa"}}>Feed</a>
      <a href="#" onClick={() => setPage("events")} style={{color: page==="events"?"#ffd700":"#aaa"}}>Events</a>
      <a href="#" onClick={() => setPage("profile")} style={{color: page==="profile"?"#ffd700":"#aaa"}}>Profile</a>
      {user === "admin" && <a href="#" onClick={() => setPage("admin")} style={{color: page==="admin"?"#ffd700":"#f0a"}}>⚙ Backstage</a>}
      <span style={{marginLeft:"auto",fontSize:"0.9em"}}>{user}</span>
      <a href="#" onClick={onLogout} style={{color:"#f88"}}>Logout</a>
    </nav>
  );
}

function Feed() {
  const [posts, setPosts] = React.useState([]);
  const [text, setText] = React.useState("");
  const [err, setErr] = React.useState("");

  function load() {
    fetch("posts.tl").then(r => r.json()).then(setPosts).catch(() => {});
  }
  React.useEffect(load, []);

  function send(e) {
    e.preventDefault();
    if (!text.trim()) return;
    setErr("");
    postForm("posts.tl", {message: text}).then((d) => {
      if (d.ok) { setText(""); load(); }
      else setErr(d.error || "Failed to post");
    }).catch(() => setErr("Network error"));
  }

  return (
    <div>
      <h2>Cat Feed 🐾</h2>
      <form onSubmit={send} style={{display:"flex",gap:"0.5em",marginBottom:"1em"}}>
        <input value={text} onChange={e => setText(e.target.value)} placeholder="What's on your mind, kitty?" style={{flex:1}} />
        <button type="submit" style={{width:"auto",padding:"0.5em 1.2em"}}>Post</button>
      </form>
      {err && <p style={{color:"#c00",fontSize:"0.9em"}}>{err}</p>}
      <div>
        {posts.slice().reverse().map((p, i) => (
          <div key={i} style={{background:"#fff",padding:"0.8em",borderRadius:"6px",marginBottom:"0.5em",boxShadow:"0 1px 3px rgba(0,0,0,0.08)"}}>
            <strong>{p.user}</strong> <span style={{color:"#999",fontSize:"0.8em"}}>{timeAgo(p.time)}</span>
            <p style={{margin:"0.3em 0 0"}}>{p.text}</p>
          </div>
        ))}
        {posts.length === 0 && <p style={{color:"#999"}}>No posts yet. Be the first cat to speak up!</p>}
      </div>
    </div>
  );
}

function Events() {
  const [events, setEvents] = React.useState([]);
  const [title, setTitle] = React.useState("");
  const [when, setWhen] = React.useState("");
  const [where, setWhere] = React.useState("");
  const [adding, setAdding] = React.useState(false);

  function load() {
    fetch("events.tl").then(r => r.json()).then(setEvents);
  }
  React.useEffect(load, []);

  function create(e) {
    e.preventDefault();
    if (!title.trim()) return;
    postForm("events.tl", {title, when, where}).then((d) => {
      if (d.ok) { setTitle(""); setWhen(""); setWhere(""); setAdding(false); load(); }
    });
  }

  return (
    <div>
      <div style={{display:"flex",justifyContent:"space-between",alignItems:"center"}}>
        <h2>Cat Events 📅</h2>
        <button onClick={() => setAdding(!adding)} style={{width:"auto",padding:"0.4em 1em",fontSize:"0.9em"}}>
          {adding ? "Cancel" : "+ New Event"}
        </button>
      </div>
      {adding && (
        <form onSubmit={create} style={{background:"#fff",padding:"1em",borderRadius:"6px",marginBottom:"1em",boxShadow:"0 1px 3px rgba(0,0,0,0.08)"}}>
          <input value={title} onChange={e => setTitle(e.target.value)} placeholder="Event title" required />
          <input value={when} onChange={e => setWhen(e.target.value)} placeholder="When (e.g. Saturday 3pm)" />
          <input value={where} onChange={e => setWhere(e.target.value)} placeholder="Where (e.g. The garden)" />
          <button type="submit">Create Event</button>
        </form>
      )}
      <div>
        {events.map((ev, i) => (
          <div key={i} style={{background:"#fff",padding:"0.8em",borderRadius:"6px",marginBottom:"0.5em",boxShadow:"0 1px 3px rgba(0,0,0,0.08)"}}>
            <strong>{ev.title}</strong> <span style={{color:"#999",fontSize:"0.8em"}}>by {ev.user}</span>
            {ev.when && <p style={{margin:"0.2em 0 0",fontSize:"0.9em"}}>🕐 {ev.when}</p>}
            {ev.where && <p style={{margin:"0.2em 0 0",fontSize:"0.9em"}}>📍 {ev.where}</p>}
          </div>
        ))}
        {events.length === 0 && <p style={{color:"#999"}}>No events yet. Plan a cat meetup!</p>}
      </div>
    </div>
  );
}

function Profile({ user, profile, onUpdate }) {
  const [emoji, setEmoji] = React.useState(profile.emoji || "🐱");
  const [breed, setBreed] = React.useState(profile.breed || "");
  const [bio, setBio] = React.useState(profile.bio || "");
  const [msg, setMsg] = React.useState("");
  const [passkeyMsg, setPasskeyMsg] = React.useState("");

  function save(e) {
    e.preventDefault();
    postForm("prof.tl", {emoji, breed, bio}).then((d) => {
      if (d.ok) { setMsg("Profile saved!"); onUpdate({emoji, breed, bio}); }
    });
  }

  function doPasskeyRegister() {
    setPasskeyMsg("");
    if (!window.PublicKeyCredential) { setPasskeyMsg("WebAuthn not supported (requires HTTPS)"); return; }
    fetch("wa_regbeg.tl", {method: "POST"}).then(r => r.json())
      .then((opts) => {
        if (opts.error) { setPasskeyMsg(opts.error); return Promise.reject(); }
        opts.publicKey.challenge = base64URLToBuffer(opts.publicKey.challenge);
        opts.publicKey.user.id = base64URLToBuffer(opts.publicKey.user.id);
        if (opts.publicKey.excludeCredentials) {
          opts.publicKey.excludeCredentials = opts.publicKey.excludeCredentials.map(c => ({...c, id: base64URLToBuffer(c.id)}));
        }
        return navigator.credentials.create(opts);
      })
      .then((cred) => {
        if (!cred) return;
        return fetch("wa_regend.tl", {
          method: "POST", headers: {"Content-Type": "application/json"},
          body: JSON.stringify({
            id: cred.id, rawId: bufferToBase64URL(cred.rawId), type: cred.type,
            response: {
              attestationObject: bufferToBase64URL(cred.response.attestationObject),
              clientDataJSON: bufferToBase64URL(cred.response.clientDataJSON)
            }
          })
        }).then(r => r.json());
      })
      .then((d) => { if (d && d.ok) setPasskeyMsg("Passkey registered! 🔑"); else if (d) setPasskeyMsg(d.error || "Failed"); })
      .catch((e) => { if (e) setPasskeyMsg(e.message || "Passkey registration cancelled"); });
  }

  return (
    <div>
      <h2>Cat Profile {emoji}</h2>
      <form onSubmit={save} style={{background:"#fff",padding:"1em",borderRadius:"6px",boxShadow:"0 1px 3px rgba(0,0,0,0.08)"}}>
        <label style={{fontSize:"0.9em",color:"#666"}}>Your cat emoji</label>
        <input value={emoji} onChange={e => setEmoji(e.target.value)} placeholder="🐱" />
        <label style={{fontSize:"0.9em",color:"#666"}}>Breed</label>
        <input value={breed} onChange={e => setBreed(e.target.value)} placeholder="e.g. Maine Coon" />
        <label style={{fontSize:"0.9em",color:"#666"}}>Bio</label>
        <input value={bio} onChange={e => setBio(e.target.value)} placeholder="Tell us about yourself..." />
        <button type="submit">Save Profile</button>
        {msg && <p style={{color:"#090",fontSize:"0.9em",marginTop:"0.5em"}}>{msg}</p>}
      </form>
      <div style={{marginTop:"1em",background:"#fff",padding:"1em",borderRadius:"6px",boxShadow:"0 1px 3px rgba(0,0,0,0.08)"}}>
        <h3 style={{fontSize:"1em"}}>Passkey Authentication 🔑</h3>
        <p style={{fontSize:"0.85em",color:"#666",margin:"0.3em 0 0.8em"}}>Register a passkey for passwordless login</p>
        <button onClick={doPasskeyRegister}>Register Passkey</button>
        {passkeyMsg && <p style={{color:"#090",fontSize:"0.9em",marginTop:"0.5em"}}>{passkeyMsg}</p>}
      </div>
    </div>
  );
}

function Admin() {
  const [info, setInfo] = React.useState(null);
  const [err, setErr] = React.useState("");

  React.useEffect(() => {
    fetch("admin.tl").then(r => r.json()).then((d) => {
      if (d.error) setErr(d.error); else setInfo(d);
    }).catch(() => setErr("Failed to load"));
  }, []);

  if (err) return <p style={{color:"#c00"}}>{err}</p>;
  if (!info) return <p>Loading...</p>;

  return (
    <div>
      <h2>⚙ Backstage</h2>
      <div style={{background:"#fff",padding:"1em",borderRadius:"6px",marginBottom:"1em",boxShadow:"0 1px 3px rgba(0,0,0,0.08)"}}>
        <h3 style={{fontSize:"1em",marginBottom:"0.5em"}}>Server</h3>
        <pre style={{background:"#f8f8f8",padding:"0.8em",borderRadius:"4px",fontSize:"0.8em",overflow:"auto",whiteSpace:"pre-wrap"}}>{info.server}</pre>
        <p style={{fontSize:"0.85em",color:"#666",marginTop:"0.5em"}}>Version: {info.ver}</p>
      </div>
      <div style={{background:"#fff",padding:"1em",borderRadius:"6px",boxShadow:"0 1px 3px rgba(0,0,0,0.08)"}}>
        <h3 style={{fontSize:"1em",marginBottom:"0.5em"}}>Registered Users ({info.users ? info.users.length : 0})</h3>
        {info.users && info.users.map((u, i) => (
          <div key={i} style={{padding:"0.3em 0",borderBottom:"1px solid #eee",fontSize:"0.9em"}}>{u}</div>
        ))}
      </div>
    </div>
  );
}

function App() {
  const [page, setPage] = React.useState("login");
  const [username, setUsername] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [email, setEmail] = React.useState("");
  const [msg, setMsg] = React.useState("");
  const [user, setUser] = React.useState("");
  const [profile, setProfile] = React.useState({});
  const [authView, setAuthView] = React.useState("login");

  React.useEffect(() => {
    fetch("status.tl").then(r => r.json()).then((d) => {
      if (d.username) { setUser(d.username); setProfile(d.profile || {}); setPage("feed"); }
    });
  }, []);

  function doRegister(e) {
    e.preventDefault();
    postForm("reg.tl", {username, password, email}).then((d) => {
      if (d.ok) { setMsg("Registered! Please log in."); setAuthView("login"); }
      else setMsg(d.error);
    });
  }

  function doLogin(e) {
    e.preventDefault();
    postForm("login.tl", {username, password}).then((d) => {
      if (d.ok) { setUser(username); setMsg(""); setPage("feed"); }
      else setMsg(d.error);
    });
  }

  function doPasskeyLogin(e) {
    e.preventDefault();
    if (!username) { setMsg("Enter your username first"); return; }
    postForm("wa_logbeg.tl", {username})
      .then((opts) => {
        if (opts.error) { setMsg(opts.error); return Promise.reject(); }
        opts.publicKey.challenge = base64URLToBuffer(opts.publicKey.challenge);
        if (opts.publicKey.allowCredentials) {
          opts.publicKey.allowCredentials = opts.publicKey.allowCredentials.map(c => ({...c, id: base64URLToBuffer(c.id)}));
        }
        return navigator.credentials.get(opts);
      })
      .then((a) => {
        if (!a) return;
        return fetch("wa_logend.tl?username=" + encodeURIComponent(username), {
          method: "POST", headers: {"Content-Type": "application/json"},
          body: JSON.stringify({
            id: a.id, rawId: bufferToBase64URL(a.rawId), type: a.type,
            response: {
              authenticatorData: bufferToBase64URL(a.response.authenticatorData),
              clientDataJSON: bufferToBase64URL(a.response.clientDataJSON),
              signature: bufferToBase64URL(a.response.signature),
              userHandle: a.response.userHandle ? bufferToBase64URL(a.response.userHandle) : ""
            }
          })
        }).then(r => r.json());
      })
      .then((d) => {
        if (d && d.ok) { setUser(username); setMsg(""); setPage("feed"); }
        else if (d) setMsg(d.error || "Passkey login failed");
      })
      .catch(() => {});
  }

  function doLogout() {
    postForm("logout.tl", {}).then(() => { setUser(""); setProfile({}); setMsg(""); setPage("login"); });
  }

  if (page === "login") {
    return (
      <div style={{display:"flex",justifyContent:"center",alignItems:"center",minHeight:"100vh"}}>
        <div className="card">
          <h1>🐱 Catbook</h1>
          <p style={{color:"#999",marginBottom:"1em",fontSize:"0.9em"}}>A social network for cats</p>
          <h2 style={{fontSize:"1.1em"}}>{authView === "login" ? "Log in" : "Register"}</h2>
          {msg && <p className="msg">{msg}</p>}
          <form onSubmit={authView === "login" ? doLogin : doRegister}>
            <input placeholder="Username" value={username} onChange={e => setUsername(e.target.value)} required />
            <input placeholder="Password" type="password" value={password} onChange={e => setPassword(e.target.value)} required />
            {authView === "register" && <input placeholder="Email" type="email" value={email} onChange={e => setEmail(e.target.value)} required />}
            <button type="submit">{authView === "login" ? "Log in" : "Register"}</button>
          </form>
          {authView === "login" && (
            <button onClick={doPasskeyLogin} style={{marginTop:"0.5em",background:"#333"}}>🔑 Sign in with passkey</button>
          )}
          <p style={{marginTop:"0.8em"}}>
            {authView === "login" ? (
              <a href="#" onClick={() => { setAuthView("register"); setMsg(""); }}>Create an account</a>
            ) : (
              <a href="#" onClick={() => { setAuthView("login"); setMsg(""); }}>Back to login</a>
            )}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div style={{minHeight:"100vh",background:"#f0f0f5"}}>
      <Nav page={page} setPage={setPage} user={user} onLogout={doLogout} />
      <div style={{maxWidth:"600px",margin:"1.5em auto",padding:"0 1em"}}>
        {page === "feed" && <Feed />}
        {page === "events" && <Events />}
        {page === "profile" && <Profile user={user} profile={profile} onUpdate={setProfile} />}
        {page === "admin" && <Admin />}
      </div>
    </div>
  );
}

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(<App />);
