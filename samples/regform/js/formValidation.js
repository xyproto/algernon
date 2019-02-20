// Find all the elements that may be involved in a register or login form
var usernameField = document.getElementById("username");
var password1Field = document.getElementById("password1");
var password2Field = document.getElementById("password2");
var emailField = document.getElementById("email");
var registerButton = document.getElementById("registerButton");
var usernameStar = document.getElementById("usernameStar");
var password1Star = document.getElementById("password1Star");
var password2Star = document.getElementById("password2Star");
var emailStar = document.getElementById("emailStar");

// Hide all status icons
if (usernameStar != null) {
  usernameStar.style.visibility = "hidden";
}
if (password1Star != null) {
  password1Star.style.visibility = "hidden";
}
if (password2Star != null) {
  password2Star.style.visibility = "hidden";
}
if (emailStar != null) {
  emailStar.style.visibility = "hidden";
}

// validEmail checks if a given email address appears to be valid.
// https://stackoverflow.com/a/46181/131264
function validEmail(email) {
  var re = /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/;
  return re.test(String(email).toLowerCase());
}

// focusError places the focus on the first invalid field,
// and also colors the border around it red by adding the is-error class.
function focusError(onlyUsernameAndPassword) {
  // Validate the username and first password field for both the registration
  // and the login form.
  if (usernameField.getAttribute('valid') != "true") {
    usernameField.classList.add('is-error');
    usernameField.classList.remove('is-success');
    usernameField.focus();
    return true;
  }
  if (password1Field.getAttribute('valid') != "true") {
    password1Field.classList.add('is-error');
    password1Field.classList.remove('is-success');
    password1Field.focus();
    return true;
  }
  // Validate a second password field and email field only for the
  // registration form.
  if (!onlyUsernameAndPassword) {
    if (password2Field.getAttribute('valid') != "true") {
      password2Field.classList.add('is-error');
      password2Field.classList.remove('is-success');
      password2Field.focus();
      return true;
    }
    if (emailField.getAttribute('valid') != "true") {
      emailField.classList.add('is-error');
      emailField.classList.remove('is-success');
      emailField.focus();
      return true;
    }
  }
  return false;
}

/* Make an icon visible when focus is lost in the user field,
 * and there is something there. Assume usernameField is not null.
 */
usernameField.oninput = function() {
  if (usernameField.value.length > 0) {
    usernameStar.style.visibility = "visible";
    usernameField.classList.remove('is-error');
    usernameField.classList.add('is-success');
    usernameField.setAttribute('valid', true);
  } else {
    usernameStar.style.visibility = "hidden";
    usernameField.classList.remove('is-success');
    usernameField.setAttribute('valid', false);
  }
};
usernameField.onblur = function() { usernameField.classList.remove('is-success'); usernameField.oninput(); };
usernameField.onfocus = function() { usernameField.classList.add('is-success'); usernameField.oninput(); };

/* Make an icon visible when focus is lost in the password field,
 * and there is something there. Assume password1Field is not null.
 */
password1Field.oninput = function() {
  if (password1Field.value.length > 0) {
    password1Star.style.visibility = "visible";
    password1Field.classList.remove('is-error');
    password1Field.classList.add('is-success');
    password1Field.setAttribute('valid', true);
  } else {
    password1Star.style.visibility = "hidden";
    password1Field.classList.remove('is-success');
    password1Field.setAttribute('valid', false);
  }
  // Also evaluate the validity of password2, if available
  if (password2Field != null) {
    password2Field.onblur();
  }
};
password1Field.onblur = function() { password1Field.classList.remove('is-success'); password1Field.oninput(); };
password1Field.onfocus = function() { password1Field.classList.add('is-success'); password1Field.oninput(); };

if (password2Field != null) {
  /* Make an icon visible when focus is lost in the password2 field,
   * but make it a cross if the passwords doesn't match.
   * Also react on keypress.
   */
  password2Field.oninput = function() {
    if ((password1Field.value == password2Field.value) && (password2Field.value.length > 0)) {
      password2Field.classList.remove('is-error');
      password2Field.classList.add('is-success');
      password2Field.setAttribute('valid', true);
    } else {
      password2Field.classList.remove('is-success');
      password2Field.setAttribute('valid', false);
    }
    // Display a status icon next to the field, if it's non-empty
    if (password2Field.value.length > 0) {
      password2Star.style.visibility = "visible";
    } else {
      password2Star.style.visibility = "hidden";
    }
    // Set the status icon to either thumbs up or an "X"
    if (password1Field.value == password2Field.value) {
      password2Star.classList.remove('close');
      password2Star.classList.add('like');
    } else {
      password2Star.classList.remove('like');
      password2Star.classList.add('close');
    }
  };
  password2Field.onblur = function() { password2Field.classList.remove('is-success'); password2Field.oninput(); };
  password2Field.onfocus = function() { password2Field.classList.add('is-success'); password2Field.oninput(); };
}

if (emailField != null) {
  /* Make an icon visible when focus is lost from the email field,
   * if there is something there.
   */
  emailField.oninput = function() {
    if ((emailField.value.length > 0) && validEmail(emailField.value)) {
      emailStar.style.visibility = "visible";
      emailField.classList.remove('is-error');
      emailField.classList.add('is-success');
      emailField.setAttribute('valid', true);
    } else {
      emailStar.style.visibility = "hidden";
      emailField.classList.remove('is-success');
      emailField.setAttribute('valid', false);
    }
  };
  emailField.onblur = function() { emailField.classList.remove('is-success'); emailField.oninput(); };
  emailField.onfocus = function() { emailField.classList.add('is-success'); emailField.oninput(); };
}

usernameField.focus();

// Enable the registration button, if available
var registerButton = document.getElementById("registerButton");
if (registerButton != null) {
  var currentScript = document.currentScript;
  registerButton.onclick = function() {
    var registrationURL = "/register/";
    if (currentScript != null && currentScript.getAttribute('registrationURL') != null) {
      registrationURL = currentScript.getAttribute('registrationURL');
    }
    document.registerForm.setAttribute('action', registrationURL + '?username=' + usernameField.value);
    // Focus fields that are invalid and return false to not submit the form,
    // or return true to submit the form. "false" is because it's a register form validation.
    return !focusError(false);
  };
}

// Enable the login button, if available
var loginButton = document.getElementById("loginButton");
if (loginButton != null) {
  var currentScript = document.currentScript;
  loginButton.onclick = function() {
    var loginURL = "/login/";
    if (currentScript != null && currentScript.getAttribute('loginURL') != null) {
      loginURL = currentScript.getAttribute('loginURL');
    }
    document.loginForm.setAttribute('action', loginURL + '?username=' + usernameField.value);
    //
    // Focus fields that are invalid and return false to not submit the form,
    // or return true to submit the form. "true" is because it's only a login form validation.
    return !focusError(true);
  };
}
