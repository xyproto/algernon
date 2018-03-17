// tutorial1.jsx

// From tutorial1.js
var CommentBox = React.createClass({
  render: function() {
    return (
      <div className="commentBox">
        Hello, world! I am a CommentBox.
      </div>
    );
  }
});

// Render the CommentBox element
React.render(
  <CommentBox />,
  document.getElementById('content')
);
