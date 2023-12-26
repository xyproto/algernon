class CommentBox extends React.Component {
  constructor(props) {
    super(props);
    this.state = { data: [] };
  }

  loadCommentsFromServer = () => {
    $.ajax({
      url: this.props.url,
      dataType: 'json',
      cache: false,
      success: (data) => {
        this.setState({ data: data });
      },
      error: (xhr, status, err) => {
        console.error(this.props.url, status, err.toString());
      }
    });
  }

  handleCommentSubmit = (comment) => {
    var comments = this.state.data;
    var newComments = comments.concat([comment]);
    this.setState({ data: newComments });
    $.ajax({
      url: this.props.url,
      dataType: 'json',
      type: 'POST',
      data: comment,
      success: (data) => {
        this.setState({ data: data });
      },
      error: (xhr, status, err) => {
        console.error(this.props.url, status, err.toString());
      }
    });
  }

  componentDidMount() {
    this.loadCommentsFromServer();
    this.interval = setInterval(this.loadCommentsFromServer, this.props.pollInterval);
  }

  componentWillUnmount() {
    clearInterval(this.interval);
  }

  render() {
    return (
      <div className="commentBox">
        <h1>Comments</h1>
        <CommentList data={this.state.data} />
        <CommentForm onCommentSubmit={this.handleCommentSubmit} />
      </div>
    );
  }
}

class CommentForm extends React.Component {
  constructor(props) {
    super(props);
    this.authorRef = React.createRef();
    this.textRef = React.createRef();
  }

  handleSubmit = (e) => {
    e.preventDefault();
    const author = this.authorRef.current.value.trim();
    const text = this.textRef.current.value.trim();
    if (!text || !author) {
      return;
    }
    this.props.onCommentSubmit({ author: author, text: text });
    this.authorRef.current.value = '';
    this.textRef.current.value = '';
    this.authorRef.current.focus();
  }

  render() {
    return (
      <form className="commentForm" onSubmit={this.handleSubmit}>
        <input type="text" placeholder="Your name" ref={this.authorRef} />
        <input type="text" placeholder="Say something..." ref={this.textRef} />
        <input type="submit" value="Post" />
      </form>
    );
  }
}

class CommentList extends React.Component {
  render() {
    var commentNodes = this.props.data.map((comment, index) => (
      <Comment key={index} author={comment.author}>
        {comment.text}
      </Comment>
    ));
    return (
      <div className="commentList">
        {commentNodes}
      </div>
    );
  }
}

class Comment extends React.Component {
  render() {
    const rawMarkup = marked.parse(this.props.children.toString());
    return (
      <div className="comment">
        <h2 className="commentAuthor">
          {this.props.author}
        </h2>
        <span dangerouslySetInnerHTML={{ __html: rawMarkup }} />
      </div>
    );
  }
}

ReactDOM.render(
  <CommentBox url="jsondb.lua" pollInterval={2000} />,
  document.getElementById('content')
);
