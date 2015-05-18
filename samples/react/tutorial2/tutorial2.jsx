// tutorial2.jsx

// From tutorial2.js
var CommentList = React.createClass({
  render: function() {
    return (
      <div className="commentList">
        Hello, world! I am a CommentList.
      </div>
    );
  }
});

var CommentForm = React.createClass({
  render: function() {
    return (
      <div className="commentForm">
        Hello, world! I am a CommentForm.
      </div>
    );
  }
});

// Render the CommentList element
React.render(
  <CommentList />,
  document.getElementById('content1')
);

// Render the CommentForm element
React.render(
  <CommentForm />,
  document.getElementById('content2')
);
