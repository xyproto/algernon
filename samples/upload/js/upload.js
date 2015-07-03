function fileSelected(fieldID) {
  var file = document.getElementById(fieldID).files[0];
  var KiB = 1024
  var MiB = KiB * KiB
  if (file) {
    var fileSize = 0;
    if (file.size >= MiB) {
      fileSize = (Math.round(file.size * 100 / MiB) / 100).toString() + 'MiB';
    } else {
      fileSize = (Math.round(file.size * 100 / KiB) / 100).toString() + 'KiB';
    }
    document.getElementById('fileName').innerHTML = 'Name: ' + file.name;
    document.getElementById('fileSize').innerHTML = 'Size: ' + fileSize;
    document.getElementById('fileType').innerHTML = 'Type: ' + file.type;
  }
}

function uploadFile() {
  var xhr = new XMLHttpRequest();
  var fd = new FormData(document.getElementById('upload'));

  xhr.upload.addEventListener("progress", uploadProgress, false);
  xhr.addEventListener("load", uploadComplete, false);
  xhr.addEventListener("error", uploadFailed, false);
  xhr.addEventListener("abort", uploadCanceled, false);

  xhr.open("POST", "upload.lua");
  xhr.send(fd);
}

function uploadProgress(evt) {
  if (evt.lengthComputable) {
    var percentComplete = Math.round(evt.loaded * 100 / evt.total);
    document.getElementById('progressNumber').innerHTML = percentComplete.toString() + '%';
  } else {
    document.getElementById('progressNumber').innerHTML = 'unable to compute';
  }
}

function uploadComplete(evt) {
  document.write(evt.target.responseText);
}

function uploadFailed(evt) {
  alert("There was an error attempting to upload the file.");
}

function uploadCanceled(evt) {
  alert("Upload was canceled by the user, or the browser dropped the connection.");
}

