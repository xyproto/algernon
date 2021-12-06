---
--- Example registration form, using Lua and Pongo2
---
--- This is not a complete site, just the registration and confirmation form.
--- The sendEmail function does not send any email, and the confirmation links
--- are just dummy links.
---

--- sendEmail takes an email address a subject and a message, then proceeds to send
--- the email, possibly using an external application.
function sendEmail(email, subject, msg)
  log("OUTGOING EMAIL EXAMPLE:\nTO:\t" .. email .. "\nSUBJECT:\t" .. subject .. "\nBODY:\t\n" .. msg .. "\n")

  -- This can be implemented in many different ways
  --mail = io.popen("mail -s '" .. subject .. "' " .. email, "w")
  --mail:write(msg .. "\n\4")
  --mail:close()
end

-- thanksPage presents a "thanks for registering, an email has been sent" page
-- to the user. The email argument is only for presenting information.
function thanksPage(email)
  CodeLib():import("globals")
  serve2("confirm.po2", {
    sitename = sitename,
    title = "Registration complete",
    msg = "Thanks for registering, a confirmation e-mail has been sent to " .. email .. ". Follow the link in the e-mail or paste the confirmation code here:",
  })
end

-- post handles POST requests when the registration form has been filled in
function post()
  local fields = formdata()

  local password1 = fields["password1"]
  local password2 = fields["password2"]
  local username = fields["username"]
  local email = fields["email"]
  local referer = headers()["Referer"]

  if referer == nil then
    print("Can not create a registration confirmation link because the Referer header is empty:")
    pprint(headers())
    return
  end

  -- Generate a unique registration confirmation code
  local code = GenerateUniqueConfirmationCode()

  -- Match any protocol and everything up until the first slash after ://
  local baseURL = string.match(referer, "(%a+://[^/]+)/")

  local confirmLink = baseURL .. "/examplepath/login/?code=" .. code
  local reportLink = baseURL .. "/examplepath/report/?code=" .. code

  -- Check if the user is in the system, but not has not been confirmed as a valid user
  if HasUnconfirmedUser(username) then
    msgpage("User already registered", "The user " .. username .. " has already been registered, but not been confirmed.", "style/nes.min.css")
    return
  end

  -- Check if the user is in the system already
  if HasUser(username) then
    msgpage("User already exists", "The " .. username .. " user already exists.", "style/nes.min.css")
    return
  end

  -- Register and login the admin user.
  -- This is a one-time special case, for when there is no admin user.
  if username == "admin" then
    AddUser(username, password1, email)
    SetAdminStatus(username)
    Login(username)

    -- Redirect to the first-time admin page
    log("REDIRECTING TO THE FIRST-TIME ADMIN PAGE")

    -- Redirecting to dummy URL, since this is just a registration form sample
    redirect("/examplepath/admin")
    return
  end

  -- Register an unconfirmed user
  AddUser(username, password1, email)

  -- Add the user to the list of unconfirmed users
  AddUnconfirmed(username, code)

  -- Send a confirmation email
  sendEmail(email, "Confirm your registration, " .. username, "Hello " .. username .. ",\nPlease follow this link to confirm your registration: " .. confirmLink .. "\nIf you did not request this email, please report it at this link: " .. reportLink .. "\n")

  -- Inform the user of what just happened
  thanksPage(email)
end

-- This is the main function, that handles either GET or POST requests
function main()
  if urlpath() ~= "/" then
    print()
    print[[NOTE: The current URL path is not "/"! For the default URL permissions to work, Algernon must either be run from this directory, or the URL prefixes must be configured correctly. Try running `regform.sh` from the project root.]]
  end

  if method() == "GET" then
    CodeLib():import("globals")
    serve2("regForm.po2", {sitename=sitename, regPostURL="/"})
  elseif method() == "POST" then
    post()
  end
end

-- Call the main function
main()
