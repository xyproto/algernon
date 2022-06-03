local _tl_compat; if (tonumber((_VERSION or ''):match('[%d.]*$')) or 0) < 5.3 then local p, m = pcall(require, 'compat53.module'); if p then _tl_compat = m end end; local assert = _tl_compat and _tl_compat.assert or assert; local io = _tl_compat and _tl_compat.io or io; local ipairs = _tl_compat and _tl_compat.ipairs or ipairs; local load = _tl_compat and _tl_compat.load or load; local math = _tl_compat and _tl_compat.math or math; local os = _tl_compat and _tl_compat.os or os; local package = _tl_compat and _tl_compat.package or package; local pairs = _tl_compat and _tl_compat.pairs or pairs; local string = _tl_compat and _tl_compat.string or string; local table = _tl_compat and _tl_compat.table or table; local _tl_table_unpack = unpack or table.unpack
local VERSION = "0.13.1+dev"

local tl = {TypeCheckOptions = {}, Env = {}, Symbol = {}, Result = {}, Error = {}, TypeInfo = {}, TypeReport = {}, TypeReportEnv = {}, }

tl.version = function()
   return VERSION
end

tl.warning_kinds = {
   ["unused"] = true,
   ["redeclaration"] = true,
   ["branch"] = true,
   ["hint"] = true,
   ["debug"] = true,
}

tl.typecodes = {

   NIL = 0x00000001,
   NUMBER = 0x00000002,
   BOOLEAN = 0x00000004,
   STRING = 0x00000008,
   TABLE = 0x00000010,
   FUNCTION = 0x00000020,
   USERDATA = 0x00000040,
   THREAD = 0x00000080,

   IS_TABLE = 0x00000008,
   IS_NUMBER = 0x00000002,
   IS_STRING = 0x00000004,
   LUA_MASK = 0x00000fff,

   INTEGER = 0x00010002,
   ARRAY = 0x00010008,
   RECORD = 0x00020008,
   ARRAYRECORD = 0x00030008,
   MAP = 0x00040008,
   TUPLE = 0x00080008,
   EMPTY_TABLE = 0x00000008,
   ENUM = 0x00010004,

   IS_ARRAY = 0x00010008,
   IS_RECORD = 0x00020008,

   NOMINAL = 0x10000000,
   TYPE_VARIABLE = 0x08000000,

   IS_UNION = 0x40000000,
   IS_POLY = 0x20000020,

   ANY = 0xffffffff,
   UNKNOWN = 0x80008000,
   INVALID = 0x80000000,

   IS_SPECIAL = 0x80000000,
   IS_VALID = 0x00000fff,
}

local Result = tl.Result
local Env = tl.Env
local Error = tl.Error
local CompatMode = tl.CompatMode
local TypeCheckOptions = tl.TypeCheckOptions
local LoadMode = tl.LoadMode
local LoadFunction = tl.LoadFunction
local TargetMode = tl.TargetMode
local TypeInfo = tl.TypeInfo
local TypeReport = tl.TypeReport
local TypeReportEnv = tl.TypeReportEnv
local Symbol = tl.Symbol

local TokenKind = {}

local Token = {}

do
   local LexState = {}

   local last_token_kind = {

      ["identifier"] = "identifier",
      ["got -"] = "op",

      ["got ."] = ".",
      ["got .."] = "op",
      ["got ="] = "op",
      ["got ~"] = "op",
      ["got ["] = "[",
      ["got 0"] = "number",
      ["got <"] = "op",
      ["got >"] = "op",
      ["got /"] = "op",
      ["got :"] = "op",

      ["string single"] = "$invalid_string$",
      ["string single got \\"] = "$invalid_string$",
      ["string double"] = "$invalid_string$",
      ["string double got \\"] = "$invalid_string$",
      ["string long"] = "$invalid_string$",
      ["string long got ]"] = "$invalid_string$",

      ["number dec"] = "integer",
      ["number decfloat"] = "number",
      ["number hex"] = "integer",
      ["number hexfloat"] = "number",
      ["number power"] = "number",
      ["number powersign"] = "$invalid_number$",
   }

   local keywords = {
      ["and"] = true,
      ["break"] = true,
      ["do"] = true,
      ["else"] = true,
      ["elseif"] = true,
      ["end"] = true,
      ["false"] = true,
      ["for"] = true,
      ["function"] = true,
      ["goto"] = true,
      ["if"] = true,
      ["in"] = true,
      ["local"] = true,
      ["nil"] = true,
      ["not"] = true,
      ["or"] = true,
      ["repeat"] = true,
      ["return"] = true,
      ["then"] = true,
      ["true"] = true,
      ["until"] = true,
      ["while"] = true,
   }

   local lex_any_char_states = {
      ["\""] = "string double",
      ["'"] = "string single",
      ["-"] = "got -",
      ["."] = "got .",
      ["0"] = "got 0",
      ["<"] = "got <",
      [">"] = "got >",
      ["/"] = "got /",
      [":"] = "got :",
      ["="] = "got =",
      ["~"] = "got ~",
      ["["] = "got [",
   }

   for c = string.byte("a"), string.byte("z") do
      lex_any_char_states[string.char(c)] = "identifier"
   end
   for c = string.byte("A"), string.byte("Z") do
      lex_any_char_states[string.char(c)] = "identifier"
   end
   lex_any_char_states["_"] = "identifier"

   for c = string.byte("1"), string.byte("9") do
      lex_any_char_states[string.char(c)] = "number dec"
   end

   local lex_word = {}
   for c = string.byte("a"), string.byte("z") do
      lex_word[string.char(c)] = true
   end
   for c = string.byte("A"), string.byte("Z") do
      lex_word[string.char(c)] = true
   end
   for c = string.byte("0"), string.byte("9") do
      lex_word[string.char(c)] = true
   end
   lex_word["_"] = true

   local lex_decimals = {}
   for c = string.byte("0"), string.byte("9") do
      lex_decimals[string.char(c)] = true
   end

   local lex_hexadecimals = {}
   for c = string.byte("0"), string.byte("9") do
      lex_hexadecimals[string.char(c)] = true
   end
   for c = string.byte("a"), string.byte("f") do
      lex_hexadecimals[string.char(c)] = true
   end
   for c = string.byte("A"), string.byte("F") do
      lex_hexadecimals[string.char(c)] = true
   end

   local lex_any_char_kinds = {}
   local single_char_kinds = { "[", "]", "(", ")", "{", "}", ",", "#", "`", ";" }
   for _, c in ipairs(single_char_kinds) do
      lex_any_char_kinds[c] = c
   end
   for _, c in ipairs({ "+", "*", "|", "&", "%", "^" }) do
      lex_any_char_kinds[c] = "op"
   end

   local lex_space = {}
   for _, c in ipairs({ " ", "\t", "\v", "\n", "\r" }) do
      lex_space[c] = true
   end

   local escapable_characters = {
      a = true,
      b = true,
      f = true,
      n = true,
      r = true,
      t = true,
      v = true,
      z = true,
      ["\\"] = true,
      ["\'"] = true,
      ["\""] = true,
      ["\r"] = true,
      ["\n"] = true,
   }

   local function lex_string_escape(input, i, c)
      if escapable_characters[c] then
         return 0, true
      elseif c == "x" then
         return 2, (
         lex_hexadecimals[input:sub(i + 1, i + 1)] and
         lex_hexadecimals[input:sub(i + 2, i + 2)])

      elseif c == "u" then
         if input:sub(i + 1, i + 1) == "{" then
            local p = i + 2
            if not lex_hexadecimals[input:sub(p, p)] then
               return 2, false
            end
            while true do
               p = p + 1
               c = input:sub(p, p)
               if not lex_hexadecimals[c] then
                  return p - i, c == "}"
               end
            end
         end
      elseif lex_decimals[c] then
         local len = lex_decimals[input:sub(i + 1, i + 1)] and
         (lex_decimals[input:sub(i + 2, i + 2)] and 2 or 1) or
         0
         return len, tonumber(input:sub(i, i + len)) < 256
      else
         return 0, false
      end
   end

   function tl.lex(input)
      local tokens = {}

      local state = "any"
      local fwd = true
      local y = 1
      local x = 0
      local i = 0
      local lc_open_lvl = 0
      local lc_close_lvl = 0
      local ls_open_lvl = 0
      local ls_close_lvl = 0
      local errs = {}
      local nt = 0

      local tx
      local ty
      local ti
      local in_token = false

      local function begin_token()
         tx = x
         ty = y
         ti = i
         in_token = true
      end

      local function end_token(kind, tk)
         nt = nt + 1
         tokens[nt] = {
            x = tx,
            y = ty,
            i = ti,
            tk = tk,
            kind = kind,
         }
         in_token = false
      end

      local function end_token_identifier()
         local tk = input:sub(ti, i - 1)
         nt = nt + 1
         tokens[nt] = {
            x = tx,
            y = ty,
            i = ti,
            tk = tk,
            kind = keywords[tk] and "keyword" or "identifier",
         }
         in_token = false
      end

      local function end_token_prev(kind)
         local tk = input:sub(ti, i - 1)
         nt = nt + 1
         tokens[nt] = {
            x = tx,
            y = ty,
            i = ti,
            tk = tk,
            kind = kind,
         }
         in_token = false
      end

      local function end_token_here(kind)
         local tk = input:sub(ti, i)
         nt = nt + 1
         tokens[nt] = {
            x = tx,
            y = ty,
            i = ti,
            tk = tk,
            kind = kind,
         }
         in_token = false
      end

      local function drop_token()
         in_token = false
      end

      local len = #input
      if input:sub(1, 2) == "#!" then
         i = input:find("\n")
         if not i then
            i = len + 1
         end
         y = 2
         x = 0
      end
      state = "any"

      while i <= len do
         if fwd then
            i = i + 1
            if i > len then
               break
            end
         end

         local c = input:sub(i, i)

         if fwd then
            if c == "\n" then
               y = y + 1
               x = 0
            else
               x = x + 1
            end
         else
            fwd = true
         end

         if state == "any" then
            local st = lex_any_char_states[c]
            if st then
               state = st
               begin_token()
            else
               local k = lex_any_char_kinds[c]
               if k then
                  begin_token()
                  end_token(k, c)
               elseif not lex_space[c] then
                  begin_token()
                  end_token_here("$invalid$")
                  table.insert(errs, tokens[#tokens])
               end
            end
         elseif state == "identifier" then
            if not lex_word[c] then
               end_token_identifier()
               fwd = false
               state = "any"
            end
         elseif state == "string double" then
            if c == "\\" then
               state = "string double got \\"
            elseif c == "\"" then
               end_token_here("string")
               state = "any"
            end
         elseif state == "comment short" then
            if c == "\n" then
               state = "any"
            end
         elseif state == "got =" then
            local t
            if c == "=" then
               t = "=="
            else
               t = "="
               fwd = false
            end
            end_token("op", t)
            state = "any"
         elseif state == "got ." then
            if c == "." then
               state = "got .."
            elseif lex_decimals[c] then
               state = "number decfloat"
            else
               end_token(".", ".")
               fwd = false
               state = "any"
            end
         elseif state == "got :" then
            local t
            if c == ":" then
               t = "::"
            else
               t = ":"
               fwd = false
            end
            end_token(t, t)
            state = "any"
         elseif state == "got [" then
            if c == "[" then
               state = "string long"
            elseif c == "=" then
               ls_open_lvl = ls_open_lvl + 1
            else
               end_token("[", "[")
               fwd = false
               state = "any"
               ls_open_lvl = 0
            end
         elseif state == "number dec" then
            if lex_decimals[c] then

            elseif c == "." then
               state = "number decfloat"
            elseif c == "e" or c == "E" then
               state = "number powersign"
            else
               end_token_prev("integer")
               fwd = false
               state = "any"
            end
         elseif state == "got -" then
            if c == "-" then
               state = "got --"
            else
               end_token("op", "-")
               fwd = false
               state = "any"
            end
         elseif state == "got .." then
            if c == "." then
               end_token("...", "...")
            else
               end_token("op", "..")
               fwd = false
            end
            state = "any"
         elseif state == "number hex" then
            if lex_hexadecimals[c] then

            elseif c == "." then
               state = "number hexfloat"
            elseif c == "p" or c == "P" then
               state = "number powersign"
            else
               end_token_prev("integer")
               fwd = false
               state = "any"
            end
         elseif state == "got --" then
            if c == "[" then
               state = "got --["
            else
               fwd = false
               state = "comment short"
               drop_token()
            end
         elseif state == "got 0" then
            if c == "x" or c == "X" then
               state = "number hex"
            elseif c == "e" or c == "E" then
               state = "number powersign"
            elseif lex_decimals[c] then
               state = "number dec"
            elseif c == "." then
               state = "number decfloat"
            else
               end_token_prev("integer")
               fwd = false
               state = "any"
            end
         elseif state == "got --[" then
            if c == "[" then
               state = "comment long"
            elseif c == "=" then
               lc_open_lvl = lc_open_lvl + 1
            else
               fwd = false
               state = "comment short"
               drop_token()
               lc_open_lvl = 0
            end
         elseif state == "comment long" then
            if c == "]" then
               state = "comment long got ]"
            end
         elseif state == "comment long got ]" then
            if c == "]" and lc_close_lvl == lc_open_lvl then
               drop_token()
               state = "any"
               lc_open_lvl = 0
               lc_close_lvl = 0
            elseif c == "=" then
               lc_close_lvl = lc_close_lvl + 1
            else
               state = "comment long"
               lc_close_lvl = 0
            end
         elseif state == "string double got \\" then
            local skip, valid = lex_string_escape(input, i, c)
            i = i + skip
            if not valid then
               end_token_here("$invalid_string$")
               table.insert(errs, tokens[#tokens])
            end
            x = x + skip
            state = "string double"
         elseif state == "string single" then
            if c == "\\" then
               state = "string single got \\"
            elseif c == "'" then
               end_token_here("string")
               state = "any"
            end
         elseif state == "string single got \\" then
            local skip, valid = lex_string_escape(input, i, c)
            i = i + skip
            if not valid then
               end_token_here("$invalid_string$")
               table.insert(errs, tokens[#tokens])
            end
            x = x + skip
            state = "string single"
         elseif state == "got ~" then
            local t
            if c == "=" then
               t = "~="
            else
               t = "~"
               fwd = false
            end
            end_token("op", t)
            state = "any"
         elseif state == "got <" then
            local t
            if c == "=" then
               t = "<="
            elseif c == "<" then
               t = "<<"
            else
               t = "<"
               fwd = false
            end
            end_token("op", t)
            state = "any"
         elseif state == "got >" then
            local t
            if c == "=" then
               t = ">="
            elseif c == ">" then
               t = ">>"
            else
               t = ">"
               fwd = false
            end
            end_token("op", t)
            state = "any"
         elseif state == "got /" then
            local t
            if c == "/" then
               t = "//"
            else
               t = "/"
               fwd = false
            end
            end_token("op", t)
            state = "any"
         elseif state == "string long" then
            if c == "]" then
               state = "string long got ]"
            end
         elseif state == "string long got ]" then
            if c == "]" then
               if ls_close_lvl == ls_open_lvl then
                  end_token_here("string")
                  state = "any"
                  ls_open_lvl = 0
                  ls_close_lvl = 0
               end
            elseif c == "=" then
               ls_close_lvl = ls_close_lvl + 1
            else
               state = "string long"
               ls_close_lvl = 0
            end
         elseif state == "number hexfloat" then
            if c == "p" or c == "P" then
               state = "number powersign"
            elseif not lex_hexadecimals[c] then
               end_token_prev("number")
               fwd = false
               state = "any"
            end
         elseif state == "number decfloat" then
            if c == "e" or c == "E" then
               state = "number powersign"
            elseif not lex_decimals[c] then
               end_token_prev("number")
               fwd = false
               state = "any"
            end
         elseif state == "number powersign" then
            if c == "-" or c == "+" then
               state = "number power"
            elseif lex_decimals[c] then
               state = "number power"
            else
               end_token_here("$invalid_number$")
               table.insert(errs, tokens[#tokens])
               state = "any"
            end
         elseif state == "number power" then
            if not lex_decimals[c] then
               end_token_prev("number")
               fwd = false
               state = "any"
            end
         end
      end

      if in_token then
         if last_token_kind[state] then
            end_token_prev(last_token_kind[state])
            if keywords[tokens[nt].tk] then
               tokens[nt].kind = "keyword"
            end
         else
            drop_token()
         end
      end

      table.insert(tokens, { x = x + 1, y = y, i = i, tk = "$EOF$", kind = "$EOF$" })

      return tokens, (#errs > 0) and errs
   end
end

local function binary_search(list, item, cmp)
   local len = #list
   local mid
   local s, e = 1, len
   while s <= e do
      mid = math.floor((s + e) / 2)
      local val = list[mid]
      local res = cmp(val, item)
      if res then
         if mid == len then
            return mid, val
         else
            if not cmp(list[mid + 1], item) then
               return mid, val
            end
         end
         s = mid + 1
      else
         e = mid - 1
      end
   end
end

function tl.get_token_at(tks, y, x)
   local _, found = binary_search(
   tks, nil,
   function(tk)
      return tk.y < y or
      (tk.y == y and tk.x <= x)
   end)


   if found and
      found.y == y and
      found.x <= x and x < found.x + #found.tk then

      return found.tk
   end
end

local last_typeid = 0

local function new_typeid()
   last_typeid = last_typeid + 1
   return last_typeid
end

local TypeName = {}

local table_types = {
   ["array"] = true,
   ["map"] = true,
   ["arrayrecord"] = true,
   ["record"] = true,
   ["emptytable"] = true,
}

local Type = {}

local Operator = {}

local NodeKind = {}

local FactType = {}

local Fact = {}

local KeyParsed = {}

local Node = {ExpectedContext = {}, }

local function is_array_type(t)
   return t.typename == "array" or t.typename == "arrayrecord"
end

local function is_record_type(t)
   return t.typename == "record" or t.typename == "arrayrecord"
end

local function is_number_type(t)
   return t.typename == "number" or t.typename == "integer"
end

local function is_typetype(t)
   return t.typename == "typetype" or t.typename == "nestedtype"
end

local ParseState = {}

local ParseTypeListMode = {}

local parse_type_list
local parse_expression
local parse_expression_and_tk
local parse_statements
local parse_argument_list
local parse_argument_type_list
local parse_type
local parse_newtype
local parse_enum_body
local parse_record_body

local function fail(ps, i, msg)
   if not ps.tokens[i] then
      local eof = ps.tokens[#ps.tokens]
      table.insert(ps.errs, { filename = ps.filename, y = eof.y, x = eof.x, msg = msg or "unexpected end of file" })
      return #ps.tokens
   end
   table.insert(ps.errs, { filename = ps.filename, y = ps.tokens[i].y, x = ps.tokens[i].x, msg = assert(msg, "syntax error, but no error message provided") })
   return math.min(#ps.tokens, i + 1)
end

local function end_at(node, tk)
   node.yend = tk.y
   node.xend = tk.x + #tk.tk - 1
end

local function verify_tk(ps, i, tk)
   if ps.tokens[i].tk == tk then
      return i + 1
   end
   return fail(ps, i, "syntax error, expected '" .. tk .. "'")
end

local function verify_end(ps, i, istart, node)
   if ps.tokens[i].tk == "end" then
      node.yend = ps.tokens[i].y
      node.xend = ps.tokens[i].x + 2
      return i + 1
   end
   end_at(node, ps.tokens[i])
   return fail(ps, i, "syntax error, expected 'end' to close construct started at " .. ps.filename .. ":" .. ps.tokens[istart].y .. ":" .. ps.tokens[istart].x .. ":")
end

local function new_node(tokens, i, kind)
   local t = tokens[i]
   return { y = t.y, x = t.x, tk = t.tk, kind = kind or t.kind }
end

local function a_type(t)
   t.typeid = new_typeid()
   return t
end

local function new_type(ps, i, typename)
   local token = ps.tokens[i]
   return a_type({
      typename = assert(typename),
      filename = ps.filename,
      y = token.y,
      x = token.x,
      tk = token.tk,
   })
end

local function verify_kind(ps, i, kind, node_kind)
   if ps.tokens[i].kind == kind then
      return i + 1, new_node(ps.tokens, i, node_kind)
   end
   return fail(ps, i, "syntax error, expected " .. kind)
end

local SkipFunction = {}

local function failskip(ps, i, msg, skip_fn, starti)
   local err_ps = {
      tokens = ps.tokens,
      errs = {},
      required_modules = {},
   }
   local skip_i = skip_fn(err_ps, starti or i)
   fail(ps, starti or i, msg)
   return skip_i or (i + 1)
end

local function skip_record(ps, i)
   i = i + 1
   return parse_record_body(ps, i, {}, {})
end

local function skip_enum(ps, i)
   i = i + 1
   return parse_enum_body(ps, i, {}, {})
end

local function parse_table_value(ps, i)
   local next_word = ps.tokens[i].tk
   local e
   if next_word == "record" then
      i = failskip(ps, i, "syntax error: this syntax is no longer valid; declare nested record inside a record", skip_record)
   elseif next_word == "enum" then
      i = failskip(ps, i, "syntax error: this syntax is no longer valid; declare nested enum inside a record", skip_enum)
   else
      i, e = parse_expression(ps, i)
   end
   if not e then
      e = new_node(ps.tokens, i - 1, "error_node")
   end
   return i, e
end

local function parse_table_item(ps, i, n)
   local node = new_node(ps.tokens, i, "table_item")
   if ps.tokens[i].kind == "$EOF$" then
      return fail(ps, i, "unexpected eof")
   end

   if ps.tokens[i].tk == "[" then
      node.key_parsed = "long"
      i = i + 1
      i, node.key = parse_expression_and_tk(ps, i, "]")
      i = verify_tk(ps, i, "=")
      i, node.value = parse_table_value(ps, i)
      return i, node, n
   elseif ps.tokens[i].kind == "identifier" then
      if ps.tokens[i + 1].tk == "=" then
         node.key_parsed = "short"
         i, node.key = verify_kind(ps, i, "identifier", "string")
         node.key.conststr = node.key.tk
         node.key.tk = '"' .. node.key.tk .. '"'
         i = verify_tk(ps, i, "=")
         i, node.value = parse_table_value(ps, i)
         return i, node, n
      elseif ps.tokens[i + 1].tk == ":" then
         node.key_parsed = "short"
         local orig_i = i
         local try_ps = {
            filename = ps.filename,
            tokens = ps.tokens,
            errs = {},
            required_modules = ps.required_modules,
         }
         i, node.key = verify_kind(try_ps, i, "identifier", "string")
         node.key.conststr = node.key.tk
         node.key.tk = '"' .. node.key.tk .. '"'
         i = verify_tk(try_ps, i, ":")
         i, node.decltype = parse_type(try_ps, i)
         if node.decltype and ps.tokens[i].tk == "=" then
            i = verify_tk(try_ps, i, "=")
            i, node.value = parse_table_value(try_ps, i)
            if node.value then
               for _, e in ipairs(try_ps.errs) do
                  table.insert(ps.errs, e)
               end
               return i, node, n
            end
         end

         node.decltype = nil
         i = orig_i
      end
   end

   node.key = new_node(ps.tokens, i, "integer")
   node.key_parsed = "implicit"
   node.key.constnum = n
   node.key.tk = tostring(n)
   i, node.value = parse_expression(ps, i)
   if not node.value then
      return fail(ps, i, "expected an expression")
   end
   return i, node, n + 1
end

local ParseItem = {}

local SeparatorMode = {}

local function parse_list(ps, i, list, close, sep, parse_item)
   local n = 1
   while ps.tokens[i].kind ~= "$EOF$" do
      if close[ps.tokens[i].tk] then
         end_at(list, ps.tokens[i])
         break
      end
      local item
      local oldn = n
      i, item, n = parse_item(ps, i, n)
      n = n or oldn
      table.insert(list, item)
      if ps.tokens[i].tk == "," then
         i = i + 1
         if sep == "sep" and close[ps.tokens[i].tk] then
            fail(ps, i, "unexpected '" .. ps.tokens[i].tk .. "'")
            return i, list
         end
      elseif sep == "term" and ps.tokens[i].tk == ";" then
         i = i + 1
      elseif not close[ps.tokens[i].tk] then
         local options = {}
         for k, _ in pairs(close) do
            table.insert(options, "'" .. k .. "'")
         end
         table.sort(options)
         table.insert(options, "','")
         local expected = "syntax error, expected one of: " .. table.concat(options, ", ")
         fail(ps, i, expected)
         local first = options[1]:sub(2, -2)

         if first ~= "}" and ps.tokens[i].y ~= ps.tokens[i - 1].y then

            table.insert(ps.tokens, i, { tk = first, y = ps.tokens[i - 1].y, x = ps.tokens[i - 1].x + 1, kind = "keyword" })
            return i, list
         end
      end
   end
   return i, list
end

local function parse_bracket_list(ps, i, list, open, close, sep, parse_item)
   i = verify_tk(ps, i, open)
   i = parse_list(ps, i, list, { [close] = true }, sep, parse_item)
   i = verify_tk(ps, i, close)
   return i, list
end

local function parse_table_literal(ps, i)
   local node = new_node(ps.tokens, i, "table_literal")
   return parse_bracket_list(ps, i, node, "{", "}", "term", parse_table_item)
end

local function parse_trying_list(ps, i, list, parse_item)
   local try_ps = {
      filename = ps.filename,
      tokens = ps.tokens,
      errs = {},
      required_modules = ps.required_modules,
   }
   local tryi, item = parse_item(try_ps, i)
   if not item then
      return i, list
   end
   for _, e in ipairs(try_ps.errs) do
      table.insert(ps.errs, e)
   end
   i = tryi
   table.insert(list, item)
   if ps.tokens[i].tk == "," then
      while ps.tokens[i].tk == "," do
         i = i + 1
         i, item = parse_item(ps, i)
         table.insert(list, item)
      end
   end
   return i, list
end

local function parse_typearg_type(ps, i)
   local backtick = false
   if ps.tokens[i].tk == "`" then
      i = verify_tk(ps, i, "`")
      backtick = true
   end
   i = verify_kind(ps, i, "identifier")
   return i, a_type({
      y = ps.tokens[i - 2].y,
      x = ps.tokens[i - 2].x,
      typename = "typearg",
      typearg = (backtick and "`" or "") .. ps.tokens[i - 1].tk,
   })
end

local function parse_typevar_type(ps, i)
   i = verify_tk(ps, i, "`")
   i = verify_kind(ps, i, "identifier")
   return i, a_type({
      y = ps.tokens[i - 2].y,
      x = ps.tokens[i - 2].x,
      typename = "typevar",
      typevar = "`" .. ps.tokens[i - 1].tk,
   })
end

local function parse_typearg_list(ps, i)
   if ps.tokens[i + 1].tk == ">" then
      return fail(ps, i + 1, "type argument list cannot be empty")
   end
   local typ = new_type(ps, i, "tuple")
   return parse_bracket_list(ps, i, typ, "<", ">", "sep", parse_typearg_type)
end

local function parse_typeval_list(ps, i)
   if ps.tokens[i + 1].tk == ">" then
      return fail(ps, i + 1, "type argument list cannot be empty")
   end
   local typ = new_type(ps, i, "tuple")
   return parse_bracket_list(ps, i, typ, "<", ">", "sep", parse_type)
end

local function parse_return_types(ps, i)
   return parse_type_list(ps, i, "rets")
end

local function parse_function_type(ps, i)
   local typ = new_type(ps, i, "function")
   i = i + 1
   if ps.tokens[i].tk == "<" then
      i, typ.typeargs = parse_typearg_list(ps, i)
   end
   if ps.tokens[i].tk == "(" then
      i, typ.args = parse_argument_type_list(ps, i)
      i, typ.rets = parse_return_types(ps, i)
   else
      typ.args = a_type({ typename = "tuple", is_va = true, a_type({ typename = "any" }) })
      typ.rets = a_type({ typename = "tuple", is_va = true, a_type({ typename = "any" }) })
   end
   return i, typ
end

local NIL = a_type({ typename = "nil" })
local ANY = a_type({ typename = "any" })
local TABLE = a_type({ typename = "map", keys = ANY, values = ANY })
local NUMBER = a_type({ typename = "number" })
local STRING = a_type({ typename = "string" })
local THREAD = a_type({ typename = "thread" })
local BOOLEAN = a_type({ typename = "boolean" })
local INTEGER = a_type({ typename = "integer" })

local simple_types = {
   ["nil"] = NIL,
   ["any"] = ANY,
   ["table"] = TABLE,
   ["number"] = NUMBER,
   ["string"] = STRING,
   ["thread"] = THREAD,
   ["boolean"] = BOOLEAN,
   ["integer"] = INTEGER,
}

local function parse_base_type(ps, i)
   local tk = ps.tokens[i].tk
   if ps.tokens[i].kind == "identifier" then
      local st = simple_types[tk]
      if st then
         return i + 1, st
      end
      local typ = new_type(ps, i, "nominal")
      typ.names = { tk }
      i = i + 1
      while ps.tokens[i].tk == "." do
         i = i + 1
         if ps.tokens[i].kind == "identifier" then
            table.insert(typ.names, ps.tokens[i].tk)
            i = i + 1
         else
            return fail(ps, i, "syntax error, expected identifier")
         end
      end

      if ps.tokens[i].tk == "<" then
         i, typ.typevals = parse_typeval_list(ps, i)
      end
      return i, typ
   elseif tk == "{" then
      i = i + 1
      local decl = new_type(ps, i, "array")
      local t
      i, t = parse_type(ps, i)
      if not t then
         return i
      end
      if ps.tokens[i].tk == "}" then
         decl.elements = t
         end_at(decl, ps.tokens[i])
         i = verify_tk(ps, i, "}")
      elseif ps.tokens[i].tk == "," then
         decl.typename = "tupletable"
         decl.types = { t }
         local n = 2
         repeat
            i = i + 1
            i, decl.types[n] = parse_type(ps, i)
            if not decl.types[n] then
               break
            end
            n = n + 1
         until ps.tokens[i].tk ~= ","
         end_at(decl, ps.tokens[i])
         i = verify_tk(ps, i, "}")
      elseif ps.tokens[i].tk == ":" then
         decl.typename = "map"
         i = i + 1
         decl.keys = t
         i, decl.values = parse_type(ps, i)
         if not decl.values then
            return i
         end
         end_at(decl, ps.tokens[i])
         i = verify_tk(ps, i, "}")
      else
         return fail(ps, i, "syntax error; did you forget a '}'?")
      end
      return i, decl
   elseif tk == "function" then
      return parse_function_type(ps, i)
   elseif tk == "nil" then
      return i + 1, simple_types["nil"]
   elseif tk == "table" then
      local typ = new_type(ps, i, "map")
      typ.keys = a_type({ typename = "any" })
      typ.values = a_type({ typename = "any" })
      return i + 1, typ
   elseif tk == "`" then
      return parse_typevar_type(ps, i)
   end
   return fail(ps, i, "expected a type")
end

parse_type = function(ps, i)
   if ps.tokens[i].tk == "(" then
      i = i + 1
      local t
      i, t = parse_type(ps, i)
      i = verify_tk(ps, i, ")")
      return i, t
   end

   local bt
   local istart = i
   i, bt = parse_base_type(ps, i)
   if not bt then
      return i
   end
   if ps.tokens[i].tk == "|" then
      local u = new_type(ps, istart, "union")
      u.types = { bt }
      while ps.tokens[i].tk == "|" do
         i = i + 1
         i, bt = parse_base_type(ps, i)
         if not bt then
            return i
         end
         table.insert(u.types, bt)
      end
      bt = u
   end
   return i, bt
end

parse_type_list = function(ps, i, mode)
   local list = new_type(ps, i, "tuple")

   local first_token = ps.tokens[i].tk
   if mode == "rets" or mode == "decltype" then
      if first_token == ":" then
         i = i + 1
      else
         return i, list
      end
   end

   local optional_paren = false
   if ps.tokens[i].tk == "(" then
      optional_paren = true
      i = i + 1
   end

   local prev_i = i
   i = parse_trying_list(ps, i, list, parse_type)
   if i == prev_i and ps.tokens[i].tk ~= ")" then
      fail(ps, i - 1, "expected a type list")
   end

   if mode == "rets" and ps.tokens[i].tk == "..." then
      i = i + 1
      local nrets = #list
      if nrets > 0 then
         list.is_va = true
      else
         fail(ps, i, "unexpected '...'")
      end
   end

   if optional_paren then
      i = verify_tk(ps, i, ")")
   end

   return i, list
end

local function parse_function_args_rets_body(ps, i, node)
   local istart = i - 1
   if ps.tokens[i].tk == "<" then
      i, node.typeargs = parse_typearg_list(ps, i)
   end
   i, node.args = parse_argument_list(ps, i)
   i, node.rets = parse_return_types(ps, i)
   i, node.body = parse_statements(ps, i)
   end_at(node, ps.tokens[i])
   i = verify_end(ps, i, istart, node)
   assert(node.rets.typename == "tuple")
   return i, node
end

local function parse_function_value(ps, i)
   local node = new_node(ps.tokens, i, "function")
   i = verify_tk(ps, i, "function")
   return parse_function_args_rets_body(ps, i, node)
end

local function unquote(str)
   local f = str:sub(1, 1)
   if f == '"' or f == "'" then
      return str:sub(2, -2), false
   end
   f = str:match("^%[=*%[")
   local l = #f + 1
   return str:sub(l, -l), true
end

local function parse_literal(ps, i)
   local tk = ps.tokens[i].tk
   local kind = ps.tokens[i].kind
   if kind == "identifier" then
      return verify_kind(ps, i, "identifier", "variable")
   elseif kind == "string" then
      local node = new_node(ps.tokens, i, "string")
      node.conststr, node.is_longstring = unquote(tk)
      return i + 1, node
   elseif kind == "number" or kind == "integer" then
      local n = tonumber(tk)
      local node
      i, node = verify_kind(ps, i, kind)
      node.constnum = n
      return i, node
   elseif tk == "true" then
      return verify_kind(ps, i, "keyword", "boolean")
   elseif tk == "false" then
      return verify_kind(ps, i, "keyword", "boolean")
   elseif tk == "nil" then
      return verify_kind(ps, i, "keyword", "nil")
   elseif tk == "function" then
      return parse_function_value(ps, i)
   elseif tk == "{" then
      return parse_table_literal(ps, i)
   elseif kind == "..." then
      return verify_kind(ps, i, "...")
   elseif kind == "$invalid_string$" then
      return fail(ps, i, "malformed string")
   elseif kind == "$invalid_number$" then
      return fail(ps, i, "malformed number")
   end
   return fail(ps, i, "syntax error")
end

local function node_is_require_call(n)
   if n.e1 and n.e2 and
      n.e1.kind == "variable" and n.e1.tk == "require" and
      n.e2.kind == "expression_list" and #n.e2 == 1 and
      n.e2[1].kind == "string" then

      return n.e2[1].conststr
   elseif n.op and n.op.op == "@funcall" and
      n.e1 and n.e1.tk == "pcall" and
      n.e2 and #n.e2 == 2 and
      n.e2[1].kind == "variable" and n.e2[1].tk == "require" and
      n.e2[2].kind == "string" and n.e2[2].conststr then

      return n.e2[2].conststr
   else
      return nil
   end
end

local an_operator

do
   local precedences = {
      [1] = {
         ["not"] = 11,
         ["#"] = 11,
         ["-"] = 11,
         ["~"] = 11,
      },
      [2] = {
         ["or"] = 1,
         ["and"] = 2,
         ["is"] = 3,
         ["<"] = 3,
         [">"] = 3,
         ["<="] = 3,
         [">="] = 3,
         ["~="] = 3,
         ["=="] = 3,
         ["|"] = 4,
         ["~"] = 5,
         ["&"] = 6,
         ["<<"] = 7,
         [">>"] = 7,
         [".."] = 8,
         ["+"] = 9,
         ["-"] = 9,
         ["*"] = 10,
         ["/"] = 10,
         ["//"] = 10,
         ["%"] = 10,
         ["^"] = 12,
         ["as"] = 50,
         ["@funcall"] = 100,
         ["@index"] = 100,
         ["."] = 100,
         [":"] = 100,
      },
   }

   local is_right_assoc = {
      ["^"] = true,
      [".."] = true,
   }

   local function new_operator(tk, arity, op)
      op = op or tk.tk
      return { y = tk.y, x = tk.x, arity = arity, op = op, prec = precedences[arity][op] }
   end

   an_operator = function(node, arity, op)
      return { y = node.y, x = node.x, arity = arity, op = op, prec = precedences[arity][op] }
   end

   local args_starters = {
      ["("] = true,
      ["{"] = true,
      ["string"] = true,
   }

   local E

   local function after_valid_prefixexp(ps, prevnode, i)
      return ps.tokens[i - 1].kind == ")" or
      (prevnode.kind == "op" and
      (prevnode.op.op == "@funcall" or
      prevnode.op.op == "@index" or
      prevnode.op.op == "." or
      prevnode.op.op == ":")) or

      prevnode.kind == "identifier" or
      prevnode.kind == "variable"
   end



   local function failstore(tkop, e1)
      return { y = tkop.y, x = tkop.x, kind = "paren", e1 = e1, failstore = true }
   end

   local function P(ps, i)
      if ps.tokens[i].kind == "$EOF$" then
         return i
      end
      local e1
      local t1 = ps.tokens[i]
      if precedences[1][ps.tokens[i].tk] ~= nil then
         local op = new_operator(ps.tokens[i], 1)
         i = i + 1
         local prev_i = i
         i, e1 = P(ps, i)
         if not e1 then
            fail(ps, prev_i, "expected an expression")
            return i
         end
         e1 = { y = t1.y, x = t1.x, kind = "op", op = op, e1 = e1 }
      elseif ps.tokens[i].tk == "(" then
         i = i + 1
         local prev_i = i
         i, e1 = parse_expression_and_tk(ps, i, ")")
         if not e1 then
            fail(ps, prev_i, "expected an expression")
            return i
         end
         e1 = { y = t1.y, x = t1.x, kind = "paren", e1 = e1 }
      else
         i, e1 = parse_literal(ps, i)
      end

      if not e1 then
         return i
      end

      while true do
         local tkop = ps.tokens[i]
         if tkop.kind == "," or tkop.kind == ")" then
            break
         end
         if tkop.tk == "." or tkop.tk == ":" then
            local op = new_operator(tkop, 2)

            local prev_i = i

            local key
            i = i + 1
            i, key = verify_kind(ps, i, "identifier")
            if not key then
               return i, failstore(tkop, e1)
            end

            if op.op == ":" then
               if not args_starters[ps.tokens[i].kind] then
                  fail(ps, i, "expected a function call for a method")
                  return i, failstore(tkop, e1)
               end

               if not after_valid_prefixexp(ps, e1, prev_i) then
                  fail(ps, prev_i, "cannot call a method on this expression")
                  return i, failstore(tkop, e1)
               end
            end

            e1 = { y = tkop.y, x = tkop.x, kind = "op", op = op, e1 = e1, e2 = key }
         elseif tkop.tk == "(" then
            local op = new_operator(tkop, 2, "@funcall")

            local prev_i = i

            local args = new_node(ps.tokens, i, "expression_list")
            i, args = parse_bracket_list(ps, i, args, "(", ")", "sep", parse_expression)

            if not after_valid_prefixexp(ps, e1, prev_i) then
               fail(ps, prev_i, "cannot call this expression")
               return i, failstore(tkop, e1)
            end

            e1 = { y = args.y, x = args.x, kind = "op", op = op, e1 = e1, e2 = args }

            table.insert(ps.required_modules, node_is_require_call(e1))
         elseif tkop.tk == "[" then
            local op = new_operator(tkop, 2, "@index")

            local prev_i = i

            local idx
            i = i + 1
            i, idx = parse_expression_and_tk(ps, i, "]")

            if not after_valid_prefixexp(ps, e1, prev_i) then
               fail(ps, prev_i, "cannot index this expression")
               return i, failstore(tkop, e1)
            end

            e1 = { y = tkop.y, x = tkop.x, kind = "op", op = op, e1 = e1, e2 = idx }
         elseif tkop.kind == "string" or tkop.kind == "{" then
            local op = new_operator(tkop, 2, "@funcall")

            local prev_i = i

            local args = new_node(ps.tokens, i, "expression_list")
            local argument
            if tkop.kind == "string" then
               argument = new_node(ps.tokens, i)
               argument.conststr = unquote(tkop.tk)
               i = i + 1
            else
               i, argument = parse_table_literal(ps, i)
            end

            if not after_valid_prefixexp(ps, e1, prev_i) then
               if tkop.kind == "string" then
                  fail(ps, prev_i, "cannot use a string here; if you're trying to call the previous expression, wrap it in parentheses")
               else
                  fail(ps, prev_i, "cannot use a table here; if you're trying to call the previous expression, wrap it in parentheses")
               end
               return i, failstore(tkop, e1)
            end

            table.insert(args, argument)
            e1 = { y = args.y, x = args.x, kind = "op", op = op, e1 = e1, e2 = args }

            table.insert(ps.required_modules, node_is_require_call(e1))
         elseif tkop.tk == "as" or tkop.tk == "is" then
            local op = new_operator(tkop, 2, tkop.tk)

            i = i + 1
            local cast = new_node(ps.tokens, i, "cast")
            if ps.tokens[i].tk == "(" then
               i, cast.casttype = parse_type_list(ps, i, "casttype")
            else
               i, cast.casttype = parse_type(ps, i)
            end
            if not cast.casttype then
               return i, failstore(tkop, e1)
            end
            e1 = { y = tkop.y, x = tkop.x, kind = "op", op = op, e1 = e1, e2 = cast, conststr = e1.conststr }
         else
            break
         end
      end

      return i, e1
   end

   E = function(ps, i, lhs, min_precedence)
      local lookahead = ps.tokens[i].tk
      while precedences[2][lookahead] and precedences[2][lookahead] >= min_precedence do
         local t1 = ps.tokens[i]
         local op = new_operator(t1, 2)
         i = i + 1
         local rhs
         i, rhs = P(ps, i)
         if not rhs then
            fail(ps, i, "expected an expression")
            return i
         end
         lookahead = ps.tokens[i].tk
         while precedences[2][lookahead] and ((precedences[2][lookahead] > (precedences[2][op.op])) or
            (is_right_assoc[lookahead] and (precedences[2][lookahead] == precedences[2][op.op]))) do
            i, rhs = E(ps, i, rhs, precedences[2][lookahead])
            if not rhs then
               fail(ps, i, "expected an expression")
               return i
            end
            lookahead = ps.tokens[i].tk
         end
         lhs = { y = t1.y, x = t1.x, kind = "op", op = op, e1 = lhs, e2 = rhs }
      end
      return i, lhs
   end

   parse_expression = function(ps, i)
      local lhs
      local istart = i
      i, lhs = P(ps, i)
      if lhs then
         i, lhs = E(ps, i, lhs, 0)
      end
      if lhs then
         return i, lhs, 0
      end

      if i == istart then
         i = fail(ps, i, "expected an expression")
      end
      return i
   end
end

parse_expression_and_tk = function(ps, i, tk)
   local e
   i, e = parse_expression(ps, i)
   if not e then
      e = new_node(ps.tokens, i - 1, "error_node")
   end
   if ps.tokens[i].tk == tk then
      i = i + 1
   else

      for n = 0, 19 do
         local t = ps.tokens[i + n]
         if t.kind == "$EOF$" then
            break
         end
         if t.tk == tk then
            fail(ps, i, "syntax error, expected '" .. tk .. "'")
            return i + n + 1, e
         end
      end
      i = fail(ps, i, "syntax error, expected '" .. tk .. "'")
   end
   return i, e
end

local function parse_variable_name(ps, i)
   local is_const = false
   local node
   i, node = verify_kind(ps, i, "identifier")
   if not node then
      return i
   end
   if ps.tokens[i].tk == "<" then
      i = i + 1
      local annotation
      i, annotation = verify_kind(ps, i, "identifier")
      if annotation then
         if annotation.tk == "const" then
            is_const = true
         else
            fail(ps, i, "unknown variable annotation: " .. annotation.tk)
         end
      else
         fail(ps, i, "expected a variable annotation")
      end
      i = verify_tk(ps, i, ">")
   end
   node.is_const = is_const
   return i, node
end

local function parse_argument(ps, i)
   local node
   if ps.tokens[i].tk == "..." then
      i, node = verify_kind(ps, i, "...", "argument")
   else
      i, node = verify_kind(ps, i, "identifier", "argument")
   end
   if ps.tokens[i].tk == ":" then
      i = i + 1
      local decltype

      i, decltype = parse_type(ps, i)

      if node then
         node.decltype = decltype
      end
   end
   return i, node, 0
end

parse_argument_list = function(ps, i)
   local node = new_node(ps.tokens, i, "argument_list")
   i, node = parse_bracket_list(ps, i, node, "(", ")", "sep", parse_argument)
   for a, fnarg in ipairs(node) do
      if fnarg.tk == "..." and a ~= #node then
         fail(ps, i, "'...' can only be last argument")
      end
   end
   return i, node
end

local function parse_argument_type(ps, i)
   local is_va = false
   if ps.tokens[i].kind == "identifier" and ps.tokens[i + 1].tk == ":" then
      i = i + 2
   elseif ps.tokens[i].tk == "..." then
      if ps.tokens[i + 1].tk == ":" then
         i = i + 2
         is_va = true
      else
         return fail(ps, i, "cannot have untyped '...' when declaring the type of an argument")
      end
   end

   local typ; i, typ = parse_type(ps, i)
   if typ then

      typ.is_va = is_va
   end

   return i, typ, 0
end

parse_argument_type_list = function(ps, i)
   local list = new_type(ps, i, "tuple")
   i = parse_bracket_list(ps, i, list, "(", ")", "sep", parse_argument_type)

   if list[#list] and list[#list].is_va then
      list[#list].is_va = nil
      list.is_va = true
   end
   return i, list
end

local function parse_identifier(ps, i)
   if ps.tokens[i].kind == "identifier" then
      return i + 1, new_node(ps.tokens, i, "identifier")
   end
   i = fail(ps, i, "syntax error, expected identifier")
   return i, new_node(ps.tokens, i, "error_node")
end

local function parse_local_function(ps, i)
   i = verify_tk(ps, i, "local")
   i = verify_tk(ps, i, "function")
   local node = new_node(ps.tokens, i, "local_function")
   i, node.name = parse_identifier(ps, i)
   return parse_function_args_rets_body(ps, i, node)
end

local function parse_global_function(ps, i)
   local orig_i = i
   i = verify_tk(ps, i, "function")
   local fn = new_node(ps.tokens, i, "global_function")
   local names = {}
   i, names[1] = parse_identifier(ps, i)
   while ps.tokens[i].tk == "." do
      i = i + 1
      i, names[#names + 1] = parse_identifier(ps, i)
   end
   if ps.tokens[i].tk == ":" then
      i = i + 1
      i, names[#names + 1] = parse_identifier(ps, i)
      fn.is_method = true
   end

   if #names > 1 then
      fn.kind = "record_function"
      local owner = names[1]
      owner.kind = "type_identifier"
      for i2 = 2, #names - 1 do
         local dot = an_operator(names[i2], 2, ".")
         names[i2].kind = "identifier"
         owner = { y = names[i2].y, x = names[i2].x, kind = "op", op = dot, e1 = owner, e2 = names[i2] }
      end
      fn.fn_owner = owner
   end
   fn.name = names[#names]

   local selfx, selfy = ps.tokens[i].x, ps.tokens[i].y
   i = parse_function_args_rets_body(ps, i, fn)
   if fn.is_method then
      table.insert(fn.args, 1, { x = selfx, y = selfy, tk = "self", kind = "identifier" })
   end

   if not fn.name then
      return orig_i + 1
   end

   return i, fn
end

local function parse_if_block(ps, i, n, node, is_else)
   local block = new_node(ps.tokens, i, "if_block")
   i = i + 1
   block.if_parent = node
   block.if_block_n = n
   if not is_else then
      i, block.exp = parse_expression_and_tk(ps, i, "then")
      if not block.exp then
         return i
      end
   end
   i, block.body = parse_statements(ps, i)
   if not block.body then
      return i
   end
   end_at(block.body, ps.tokens[i - 1])
   block.yend, block.xend = block.body.yend, block.body.xend
   table.insert(node.if_blocks, block)
   return i, node
end

local function parse_if(ps, i)
   local istart = i
   local node = new_node(ps.tokens, i, "if")
   node.if_blocks = {}
   i, node = parse_if_block(ps, i, 1, node)
   if not node then
      return i
   end
   local n = 2
   while ps.tokens[i].tk == "elseif" do
      i, node = parse_if_block(ps, i, n, node)
      if not node then
         return i
      end
      n = n + 1
   end
   if ps.tokens[i].tk == "else" then
      i, node = parse_if_block(ps, i, n, node, true)
      if not node then
         return i
      end
   end
   i = verify_end(ps, i, istart, node)
   return i, node
end

local function parse_while(ps, i)
   local istart = i
   local node = new_node(ps.tokens, i, "while")
   i = verify_tk(ps, i, "while")
   i, node.exp = parse_expression_and_tk(ps, i, "do")
   i, node.body = parse_statements(ps, i)
   i = verify_end(ps, i, istart, node)
   return i, node
end

local function parse_fornum(ps, i)
   local istart = i
   local node = new_node(ps.tokens, i, "fornum")
   i = i + 1
   i, node.var = parse_identifier(ps, i)
   i = verify_tk(ps, i, "=")
   i, node.from = parse_expression_and_tk(ps, i, ",")
   i, node.to = parse_expression(ps, i)
   if ps.tokens[i].tk == "," then
      i = i + 1
      i, node.step = parse_expression_and_tk(ps, i, "do")
   else
      i = verify_tk(ps, i, "do")
   end
   i, node.body = parse_statements(ps, i)
   i = verify_end(ps, i, istart, node)
   return i, node
end

local function parse_forin(ps, i)
   local istart = i
   local node = new_node(ps.tokens, i, "forin")
   i = i + 1
   node.vars = new_node(ps.tokens, i, "variable_list")
   i, node.vars = parse_list(ps, i, node.vars, { ["in"] = true }, "sep", parse_variable_name)
   i = verify_tk(ps, i, "in")
   node.exps = new_node(ps.tokens, i, "expression_list")
   i = parse_list(ps, i, node.exps, { ["do"] = true }, "sep", parse_expression)
   if #node.exps < 1 then
      return fail(ps, i, "missing iterator expression in generic for")
   elseif #node.exps > 3 then
      return fail(ps, i, "too many expressions in generic for")
   end
   i = verify_tk(ps, i, "do")
   i, node.body = parse_statements(ps, i)
   i = verify_end(ps, i, istart, node)
   return i, node
end

local function parse_for(ps, i)
   if ps.tokens[i + 1].kind == "identifier" and ps.tokens[i + 2].tk == "=" then
      return parse_fornum(ps, i)
   else
      return parse_forin(ps, i)
   end
end

local function parse_repeat(ps, i)
   local node = new_node(ps.tokens, i, "repeat")
   i = verify_tk(ps, i, "repeat")
   i, node.body = parse_statements(ps, i)
   node.body.is_repeat = true
   i = verify_tk(ps, i, "until")
   i, node.exp = parse_expression(ps, i)
   end_at(node, ps.tokens[i - 1])
   return i, node
end

local function parse_do(ps, i)
   local istart = i
   local node = new_node(ps.tokens, i, "do")
   i = verify_tk(ps, i, "do")
   i, node.body = parse_statements(ps, i)
   i = verify_end(ps, i, istart, node)
   return i, node
end

local function parse_break(ps, i)
   local node = new_node(ps.tokens, i, "break")
   i = verify_tk(ps, i, "break")
   return i, node
end

local function parse_goto(ps, i)
   local node = new_node(ps.tokens, i, "goto")
   i = verify_tk(ps, i, "goto")
   node.label = ps.tokens[i].tk
   i = verify_kind(ps, i, "identifier")
   return i, node
end

local function parse_label(ps, i)
   local node = new_node(ps.tokens, i, "label")
   i = verify_tk(ps, i, "::")
   node.label = ps.tokens[i].tk
   i = verify_kind(ps, i, "identifier")
   i = verify_tk(ps, i, "::")
   return i, node
end

local stop_statement_list = {
   ["end"] = true,
   ["else"] = true,
   ["elseif"] = true,
   ["until"] = true,
}

local stop_return_list = {
   [";"] = true,
   ["$EOF$"] = true,
}

for k, v in pairs(stop_statement_list) do
   stop_return_list[k] = v
end

local function parse_return(ps, i)
   local node = new_node(ps.tokens, i, "return")
   i = verify_tk(ps, i, "return")
   node.exps = new_node(ps.tokens, i, "expression_list")
   i = parse_list(ps, i, node.exps, stop_return_list, "sep", parse_expression)
   if ps.tokens[i].kind == ";" then
      i = i + 1
   end
   return i, node
end

local function store_field_in_record(ps, i, field_name, t, fields, field_order)
   if not fields[field_name] then
      fields[field_name] = t
      table.insert(field_order, field_name)
   else
      local prev_t = fields[field_name]
      if t.typename == "function" and prev_t.typename == "function" then
         fields[field_name] = new_type(ps, i, "poly")
         fields[field_name].types = { prev_t, t }
      elseif t.typename == "function" and prev_t.typename == "poly" then
         table.insert(prev_t.types, t)
      else
         fail(ps, i, "attempt to redeclare field '" .. field_name .. "' (only functions can be overloaded)")
         return false
      end
   end
   return true
end

local ParseBody = {}

local function parse_nested_type(ps, i, def, typename, parse_body)
   i = i + 1

   local v
   i, v = verify_kind(ps, i, "identifier", "type_identifier")
   if not v then
      return fail(ps, i, "expected a variable name")
   end

   local nt = new_node(ps.tokens, i, "newtype")
   nt.newtype = new_type(ps, i, "typetype")
   local rdef = new_type(ps, i, typename)
   local iok = parse_body(ps, i, rdef, nt)
   if iok then
      i = iok
      nt.newtype.def = rdef
   end

   store_field_in_record(ps, i, v.tk, nt.newtype, def.fields, def.field_order)
   return i
end

parse_enum_body = function(ps, i, def, node)
   local istart = i - 1
   def.enumset = {}
   while ps.tokens[i].tk ~= "$EOF$" and ps.tokens[i].tk ~= "end" do
      local item
      i, item = verify_kind(ps, i, "string", "enum_item")
      if item then
         table.insert(node, item)
         def.enumset[unquote(item.tk)] = true
      end
   end
   i = verify_end(ps, i, istart, node)
   return i, node
end

local metamethod_names = {
   ["__add"] = true,
   ["__sub"] = true,
   ["__mul"] = true,
   ["__div"] = true,
   ["__mod"] = true,
   ["__pow"] = true,
   ["__unm"] = true,
   ["__idiv"] = true,
   ["__band"] = true,
   ["__bor"] = true,
   ["__bxor"] = true,
   ["__bnot"] = true,
   ["__shl"] = true,
   ["__shr"] = true,
   ["__concat"] = true,
   ["__len"] = true,
   ["__eq"] = true,
   ["__lt"] = true,
   ["__le"] = true,
   ["__index"] = true,
   ["__newindex"] = true,
   ["__call"] = true,
   ["__tostring"] = true,
   ["__pairs"] = true,
   ["__gc"] = true,
}

parse_record_body = function(ps, i, def, node)
   local istart = i - 1
   def.fields = {}
   def.field_order = {}
   if ps.tokens[i].tk == "<" then
      i, def.typeargs = parse_typearg_list(ps, i)
   end
   while not (ps.tokens[i].kind == "$EOF$" or ps.tokens[i].tk == "end") do
      if ps.tokens[i].tk == "userdata" and ps.tokens[i + 1].tk ~= ":" then
         if def.is_userdata then
            fail(ps, i, "duplicated 'userdata' declaration in record")
         else
            def.is_userdata = true
         end
         i = i + 1
      elseif ps.tokens[i].tk == "{" then
         if def.typename == "arrayrecord" then
            i = failskip(ps, i, "duplicated declaration of array element type in record", parse_type)
         else
            i = i + 1
            local t
            i, t = parse_type(ps, i)
            if ps.tokens[i].tk == "}" then
               i = verify_tk(ps, i, "}")
            else
               return fail(ps, i, "expected an array declaration")
            end
            def.typename = "arrayrecord"
            def.elements = t
         end
      elseif ps.tokens[i].tk == "type" and ps.tokens[i + 1].tk ~= ":" then
         i = i + 1
         local iv = i
         local v
         i, v = verify_kind(ps, i, "identifier", "type_identifier")
         if not v then
            return fail(ps, i, "expected a variable name")
         end
         i = verify_tk(ps, i, "=")
         local nt
         i, nt = parse_newtype(ps, i)
         if not nt or not nt.newtype then
            return fail(ps, i, "expected a type definition")
         end

         store_field_in_record(ps, iv, v.tk, nt.newtype, def.fields, def.field_order)
      elseif ps.tokens[i].tk == "record" and ps.tokens[i + 1].tk ~= ":" then
         i = parse_nested_type(ps, i, def, "record", parse_record_body)
      elseif ps.tokens[i].tk == "enum" and ps.tokens[i + 1].tk ~= ":" then
         i = parse_nested_type(ps, i, def, "enum", parse_enum_body)
      else
         local is_metamethod = false
         if ps.tokens[i].tk == "metamethod" and ps.tokens[i + 1].tk ~= ":" then
            is_metamethod = true
            i = i + 1
         end

         local v
         if ps.tokens[i].tk == "[" then
            i, v = parse_literal(ps, i + 1)
            if v and not v.conststr then
               return fail(ps, i, "expected a string literal")
            end
            i = verify_tk(ps, i, "]")
         else
            i, v = verify_kind(ps, i, "identifier", "variable")
         end
         local iv = i
         if not v then
            return fail(ps, i, "expected a variable name")
         end

         if ps.tokens[i].tk == ":" then
            i = i + 1
            local t
            i, t = parse_type(ps, i)
            if not t then
               return fail(ps, i, "expected a type")
            end

            local field_name = v.conststr or v.tk
            local fields = def.fields
            local field_order = def.field_order
            if is_metamethod then
               if not def.meta_fields then
                  def.meta_fields = {}
                  def.meta_field_order = {}
               end
               fields = def.meta_fields
               field_order = def.meta_field_order
               if not metamethod_names[field_name] then
                  fail(ps, i - 1, "not a valid metamethod: " .. field_name)
               end
            end
            store_field_in_record(ps, iv, field_name, t, fields, field_order)
         elseif ps.tokens[i].tk == "=" then
            local next_word = ps.tokens[i + 1].tk
            if next_word == "record" or next_word == "enum" then
               return fail(ps, i, "syntax error: this syntax is no longer valid; use '" .. next_word .. " " .. v.tk .. "'")
            elseif next_word == "functiontype" then
               return fail(ps, i, "syntax error: this syntax is no longer valid; use 'type " .. v.tk .. " = function('...")
            else
               return fail(ps, i, "syntax error: this syntax is no longer valid; use 'type " .. v.tk .. " = '...")
            end
         else
            fail(ps, i, "syntax error: expected ':' for an attribute or '=' for a nested type")
         end
      end
   end
   i = verify_end(ps, i, istart, node)
   return i, node
end

parse_newtype = function(ps, i)
   local node = new_node(ps.tokens, i, "newtype")
   node.newtype = new_type(ps, i, "typetype")
   if ps.tokens[i].tk == "record" then
      local def = new_type(ps, i, "record")
      i = i + 1
      i = parse_record_body(ps, i, def, node)
      node.newtype.def = def
      return i, node
   elseif ps.tokens[i].tk == "enum" then
      local def = new_type(ps, i, "enum")
      i = i + 1
      i = parse_enum_body(ps, i, def, node)
      node.newtype.def = def
      return i, node
   else
      i, node.newtype.def = parse_type(ps, i)
      if not node.newtype.def then
         return i
      end
      return i, node
   end
   return fail(ps, i, "expected a type")
end

local function parse_assignment_expression_list(ps, i, asgn)
   asgn.exps = new_node(ps.tokens, i, "expression_list")
   repeat
      i = i + 1
      local val
      i, val = parse_expression(ps, i)
      if not val then
         if #asgn.exps == 0 then
            asgn.exps = nil
         end
         return i
      end
      table.insert(asgn.exps, val)
   until ps.tokens[i].tk ~= ","
   return i, asgn
end

local parse_call_or_assignment
do
   local function is_lvalue(node)
      return node.kind == "variable" or
      (node.kind == "op" and (node.op.op == "@index" or node.op.op == "."))
   end

   local function parse_variable(ps, i)
      local node
      i, node = parse_expression(ps, i)
      if not (node and is_lvalue(node)) then
         return fail(ps, i, "expected a variable")
      end
      return i, node
   end

   parse_call_or_assignment = function(ps, i)
      local exp
      local istart = i
      i, exp = parse_expression(ps, i)
      if not exp then
         return i
      end

      if (exp.op and exp.op.op == "@funcall") or exp.failstore then
         return i, exp
      end

      if not is_lvalue(exp) then
         return fail(ps, i, "syntax error")
      end

      local asgn = new_node(ps.tokens, istart, "assignment")
      asgn.vars = new_node(ps.tokens, istart, "variable_list")
      asgn.vars[1] = exp
      if ps.tokens[i].tk == "," then
         i = i + 1
         i = parse_trying_list(ps, i, asgn.vars, parse_variable)
         if #asgn.vars < 2 then
            return fail(ps, i, "syntax error")
         end
      end

      if ps.tokens[i].tk ~= "=" then
         verify_tk(ps, i, "=")
         return i
      end

      i, asgn = parse_assignment_expression_list(ps, i, asgn)
      return i, asgn
   end
end

local function parse_variable_declarations(ps, i, node_name)
   local asgn = new_node(ps.tokens, i, node_name)

   asgn.vars = new_node(ps.tokens, i, "variable_list")
   i = parse_trying_list(ps, i, asgn.vars, parse_variable_name)
   if #asgn.vars == 0 then
      return fail(ps, i, "expected a local variable definition")
   end

   i, asgn.decltype = parse_type_list(ps, i, "decltype")

   if ps.tokens[i].tk == "=" then

      local next_word = ps.tokens[i + 1].tk
      if next_word == "record" then
         local scope = node_name == "local_declaration" and "local" or "global"

         return failskip(ps, i + 1, "syntax error: this syntax is no longer valid; use '" .. scope .. " record " .. asgn.vars[1].tk .. "'", skip_record)
      elseif next_word == "enum" then
         local scope = node_name == "local_declaration" and "local" or "global"
         return failskip(ps, i + 1, "syntax error: this syntax is no longer valid; use '" .. scope .. " enum " .. asgn.vars[1].tk .. "'", skip_enum)
      elseif next_word == "functiontype" then
         local scope = node_name == "local_declaration" and "local" or "global"
         return failskip(ps, i + 1, "syntax error: this syntax is no longer valid; use '" .. scope .. " type " .. asgn.vars[1].tk .. " = function('...", parse_function_type)
      end

      i, asgn = parse_assignment_expression_list(ps, i, asgn)
   end
   return i, asgn
end

local function parse_type_declaration(ps, i, node_name)
   i = i + 2

   local asgn = new_node(ps.tokens, i, node_name)
   i, asgn.var = parse_variable_name(ps, i)
   if not asgn.var then
      return fail(ps, i, "expected a type name")
   end
   i = verify_tk(ps, i, "=")
   i, asgn.value = parse_newtype(ps, i)
   if not asgn.value then
      return i
   end
   if not asgn.value.newtype.def.names then
      asgn.value.newtype.def.names = { asgn.var.tk }
   end

   return i, asgn
end

local function parse_type_constructor(ps, i, node_name, type_name, parse_body)
   local asgn = new_node(ps.tokens, i, node_name)
   local nt = new_node(ps.tokens, i, "newtype")
   asgn.value = nt
   nt.newtype = new_type(ps, i, "typetype")
   local def = new_type(ps, i, type_name)
   nt.newtype.def = def

   i = i + 2

   i, asgn.var = verify_kind(ps, i, "identifier")
   if not asgn.var then
      return fail(ps, i, "expected a type name")
   end
   nt.newtype.def.names = { asgn.var.tk }

   i = parse_body(ps, i, def, nt)
   return i, asgn
end

local function skip_type_declaration(ps, i)
   return (parse_type_declaration(ps, i - 1, "local_type"))
end

local function parse_local(ps, i)
   local ntk = ps.tokens[i + 1].tk
   if ntk == "function" then
      return parse_local_function(ps, i)
   elseif ntk == "type" and ps.tokens[i + 2].kind == "identifier" then
      return parse_type_declaration(ps, i, "local_type")
   elseif ntk == "record" and ps.tokens[i + 2].kind == "identifier" then
      return parse_type_constructor(ps, i, "local_type", "record", parse_record_body)
   elseif ntk == "enum" and ps.tokens[i + 2].kind == "identifier" then
      return parse_type_constructor(ps, i, "local_type", "enum", parse_enum_body)
   end
   return parse_variable_declarations(ps, i + 1, "local_declaration")
end

local function parse_global(ps, i)
   local ntk = ps.tokens[i + 1].tk
   if ntk == "function" then
      return parse_global_function(ps, i + 1)
   elseif ntk == "type" and ps.tokens[i + 2].kind == "identifier" then
      return parse_type_declaration(ps, i, "global_type")
   elseif ntk == "record" and ps.tokens[i + 2].kind == "identifier" then
      return parse_type_constructor(ps, i, "global_type", "record", parse_record_body)
   elseif ntk == "enum" and ps.tokens[i + 2].kind == "identifier" then
      return parse_type_constructor(ps, i, "global_type", "enum", parse_enum_body)
   elseif ps.tokens[i + 1].kind == "identifier" then
      return parse_variable_declarations(ps, i + 1, "global_declaration")
   end
   return parse_call_or_assignment(ps, i)
end

local function parse_type_statement(ps, i)
   if ps.tokens[i + 1].kind == "identifier" then
      return failskip(ps, i, "types need to be declared with 'local type' or 'global type'", skip_type_declaration)
   end
   return parse_call_or_assignment(ps, i)
end

local parse_statement_fns = {
   ["::"] = parse_label,
   ["do"] = parse_do,
   ["if"] = parse_if,
   ["for"] = parse_for,
   ["goto"] = parse_goto,
   ["type"] = parse_type_statement,
   ["local"] = parse_local,
   ["while"] = parse_while,
   ["break"] = parse_break,
   ["global"] = parse_global,
   ["repeat"] = parse_repeat,
   ["return"] = parse_return,
   ["function"] = parse_global_function,
}

parse_statements = function(ps, i, toplevel)
   local node = new_node(ps.tokens, i, "statements")
   local item
   while true do
      while ps.tokens[i].kind == ";" do
         i = i + 1
         if item then
            item.semicolon = true
         end
      end

      if ps.tokens[i].kind == "$EOF$" then
         break
      end
      if (not toplevel) and stop_statement_list[ps.tokens[i].tk] then
         break
      end


      local parse_statement_fn = parse_statement_fns[ps.tokens[i].tk] or parse_call_or_assignment
      i, item = parse_statement_fn(ps, i)


      if item then
         table.insert(node, item)
      elseif i > 1 then

         local lasty = ps.tokens[i - 1].y
         while ps.tokens[i].kind ~= "$EOF$" and ps.tokens[i].y == lasty do
            i = i + 1
         end
      end
   end

   end_at(node, ps.tokens[i])
   return i, node
end

local function clear_redundant_errors(errors)
   local redundant = {}
   local lastx, lasty = 0, 0
   for i, err in ipairs(errors) do
      err.i = i
   end
   table.sort(errors, function(a, b)
      local af = a.filename or ""
      local bf = b.filename or ""
      return af < bf or
      (af == bf and (a.y < b.y or
      (a.y == b.y and (a.x < b.x or
      (a.x == b.x and (a.i < b.i))))))
   end)
   for i, err in ipairs(errors) do
      err.i = nil
      if err.x == lastx and err.y == lasty then
         table.insert(redundant, i)
      end
      lastx, lasty = err.x, err.y
   end
   for i = #redundant, 1, -1 do
      table.remove(errors, redundant[i])
   end
end

function tl.parse_program(tokens, errs, filename)
   errs = errs or {}
   local ps = {
      tokens = tokens,
      errs = errs,
      filename = filename or "",
      required_modules = {},
   }
   local i, node = parse_statements(ps, 1, true)

   clear_redundant_errors(errs)
   return i, node, ps.required_modules
end

local VisitorCallbacks = {}

local VisitorExtraCallback = {}

local Visitor = {}

local MetaMode = {}

local function fields_of(t, meta)
   local i = 1
   local field_order = meta and t.meta_field_order or t.field_order
   local fields = meta and t.meta_fields or t.fields
   return function()
      local name = field_order[i]
      if not name then
         return nil
      end
      i = i + 1
      return name, fields[name]
   end
end

local function recurse_type(ast, visit)
   local kind = ast.typename
   local cbs = visit.cbs
   local cbkind = cbs and cbs[kind]
   do
      if cbkind then
         local cbkind_before = cbkind.before
         if cbkind_before then
            cbkind_before(ast)
         end
      else
         if cbs then
            error("internal compiler error: no visitor for " .. kind)
         end
      end
   end

   local xs = {}

   if ast.typeargs then
      for _, child in ipairs(ast.typeargs) do
         table.insert(xs, recurse_type(child, visit))
      end
   end

   for i, child in ipairs(ast) do
      xs[i] = recurse_type(child, visit)
   end

   if ast.types then
      for _, child in ipairs(ast.types) do
         table.insert(xs, recurse_type(child, visit))
      end
   end
   if ast.def then
      table.insert(xs, recurse_type(ast.def, visit))
   end
   if ast.keys then
      table.insert(xs, recurse_type(ast.keys, visit))
   end
   if ast.values then
      table.insert(xs, recurse_type(ast.values, visit))
   end
   if ast.elements then
      table.insert(xs, recurse_type(ast.elements, visit))
   end
   if ast.fields then
      for _, child in fields_of(ast) do
         table.insert(xs, recurse_type(child, visit))
      end
   end
   if ast.meta_fields then
      for _, child in fields_of(ast, "meta") do
         table.insert(xs, recurse_type(child, visit))
      end
   end
   if ast.args then
      for i, child in ipairs(ast.args) do
         if i > 1 or not ast.is_method then
            table.insert(xs, recurse_type(child, visit))
         end
      end
   end
   if ast.rets then
      for _, child in ipairs(ast.rets) do
         table.insert(xs, recurse_type(child, visit))
      end
   end
   if ast.typevals then
      for _, child in ipairs(ast.typevals) do
         table.insert(xs, recurse_type(child, visit))
      end
   end
   if ast.ktype then
      table.insert(xs, recurse_type(ast.ktype, visit))
   end
   if ast.vtype then
      table.insert(xs, recurse_type(ast.vtype, visit))
   end

   local ret
   do
      local cbkind_after = cbkind and cbkind.after
      if cbkind_after then
         ret = cbkind_after(ast, xs)
      end
      local visit_after = visit.after
      if visit_after then
         ret = visit_after(ast, xs, ret)
      end
   end
   return ret
end

local function recurse_typeargs(ast, visit_type)
   if ast.typeargs then
      for _, typearg in ipairs(ast.typeargs) do
         recurse_type(typearg, visit_type)
      end
   end
end

local function extra_callback(name,
   ast,
   xs,
   visit_node)
   local cbs = visit_node.cbs
   if not cbs then return end
   local nbs = cbs[ast.kind]
   if not nbs then return end
   local bs = nbs[name]
   if not bs then return end
   bs(ast, xs)
end

local no_recurse_node = {
   ["..."] = true,
   ["nil"] = true,
   ["cast"] = true,
   ["goto"] = true,
   ["break"] = true,
   ["label"] = true,
   ["number"] = true,
   ["string"] = true,
   ["boolean"] = true,
   ["integer"] = true,
   ["variable"] = true,
   ["error_node"] = true,
   ["identifier"] = true,
   ["type_identifier"] = true,
}

local function recurse_node(root,
   visit_node,
   visit_type)
   if not root then

      return
   end

   local recurse

   local function walk_children(ast, xs)
      for i, child in ipairs(ast) do
         xs[i] = recurse(child)
      end
   end

   local function walk_vars_exps(ast, xs)
      xs[1] = recurse(ast.vars)
      if ast.decltype then
         xs[2] = recurse_type(ast.decltype, visit_type)
      end
      extra_callback("before_expressions", ast, xs, visit_node)
      if ast.exps then
         xs[3] = recurse(ast.exps)
      end
   end

   local function walk_var_value(ast, xs)
      xs[1] = recurse(ast.var)
      xs[2] = recurse(ast.value)
   end

   local function walk_named_function(ast, xs)
      recurse_typeargs(ast, visit_type)
      xs[1] = recurse(ast.name)
      xs[2] = recurse(ast.args)
      xs[3] = recurse_type(ast.rets, visit_type)
      extra_callback("before_statements", ast, xs, visit_node)
      xs[4] = recurse(ast.body)
   end

   local walkers = {
      ["op"] = function(ast, xs)
         xs[1] = recurse(ast.e1)
         local p1 = ast.e1.op and ast.e1.op.prec or nil
         if ast.op.op == ":" and ast.e1.kind == "string" then
            p1 = -999
         end
         xs[2] = p1
         if ast.op.arity == 2 then
            extra_callback("before_e2", ast, xs, visit_node)
            if ast.op.op == "is" or ast.op.op == "as" then
               xs[3] = recurse_type(ast.e2.casttype, visit_type)
            else
               xs[3] = recurse(ast.e2)
            end
            xs[4] = (ast.e2.op and ast.e2.op.prec)
         end
      end,

      ["statements"] = walk_children,
      ["argument_list"] = walk_children,
      ["table_literal"] = walk_children,
      ["variable_list"] = walk_children,
      ["expression_list"] = walk_children,

      ["table_item"] = function(ast, xs)
         xs[1] = recurse(ast.key)
         xs[2] = recurse(ast.value)
         if ast.decltype then
            xs[3] = recurse_type(ast.decltype, visit_type)
         end
      end,

      ["assignment"] = walk_vars_exps,
      ["local_declaration"] = walk_vars_exps,
      ["global_declaration"] = walk_vars_exps,

      ["local_type"] = walk_var_value,
      ["global_type"] = walk_var_value,

      ["if"] = function(ast, xs)
         for _, e in ipairs(ast.if_blocks) do
            table.insert(xs, recurse(e))
         end
      end,

      ["if_block"] = function(ast, xs)
         if ast.exp then
            xs[1] = recurse(ast.exp)
         end
         extra_callback("before_statements", ast, xs, visit_node)
         xs[2] = recurse(ast.body)
      end,

      ["while"] = function(ast, xs)
         xs[1] = recurse(ast.exp)
         extra_callback("before_statements", ast, xs, visit_node)
         xs[2] = recurse(ast.body)
      end,

      ["repeat"] = function(ast, xs)
         xs[1] = recurse(ast.body)
         xs[2] = recurse(ast.exp)
      end,

      ["function"] = function(ast, xs)
         recurse_typeargs(ast, visit_type)
         xs[1] = recurse(ast.args)
         xs[2] = recurse_type(ast.rets, visit_type)
         extra_callback("before_statements", ast, xs, visit_node)
         xs[3] = recurse(ast.body)
      end,
      ["local_function"] = walk_named_function,
      ["global_function"] = walk_named_function,
      ["record_function"] = function(ast, xs)
         recurse_typeargs(ast, visit_type)
         xs[1] = recurse(ast.fn_owner)
         xs[2] = recurse(ast.name)
         xs[3] = recurse(ast.args)
         xs[4] = recurse_type(ast.rets, visit_type)
         extra_callback("before_statements", ast, xs, visit_node)
         xs[5] = recurse(ast.body)
      end,

      ["forin"] = function(ast, xs)
         xs[1] = recurse(ast.vars)
         xs[2] = recurse(ast.exps)
         extra_callback("before_statements", ast, xs, visit_node)
         xs[3] = recurse(ast.body)
      end,

      ["fornum"] = function(ast, xs)
         xs[1] = recurse(ast.var)
         xs[2] = recurse(ast.from)
         xs[3] = recurse(ast.to)
         xs[4] = ast.step and recurse(ast.step)
         extra_callback("before_statements", ast, xs, visit_node)
         xs[5] = recurse(ast.body)
      end,

      ["return"] = function(ast, xs)
         xs[1] = recurse(ast.exps)
      end,

      ["do"] = function(ast, xs)
         xs[1] = recurse(ast.body)
      end,

      ["paren"] = function(ast, xs)
         xs[1] = recurse(ast.e1)
      end,

      ["newtype"] = function(ast, xs)
         xs[1] = recurse_type(ast.newtype, visit_type)
      end,

      ["argument"] = function(ast, xs)
         if ast.decltype then
            xs[1] = recurse_type(ast.decltype, visit_type)
         end
      end,
   }

   if not visit_node.allow_missing_cbs and not visit_node.cbs then
      error("missing cbs in visit_node")
   end
   local visit_after = visit_node.after

   recurse = function(ast)
      local xs = {}
      local kind = assert(ast.kind)

      local cbs = visit_node.cbs
      local cbkind = cbs and cbs[kind]
      do
         if cbkind then
            if cbkind.before then
               cbkind.before(ast)
            end
         else
            if cbs then
               error("internal compiler error: no visitor for " .. kind)
            end
         end
      end

      local fn = walkers[kind]
      if fn then
         fn(ast, xs)
      else
         assert(no_recurse_node[kind])
      end

      local ret
      do
         local cbkind_after = cbkind and cbkind.after
         if cbkind_after then
            ret = cbkind_after(ast, xs)
         end
         if visit_after then
            ret = visit_after(ast, xs, ret)
         end
      end
      return ret
   end

   return recurse(root)
end

local tight_op = {
   [1] = {
      ["-"] = true,
      ["~"] = true,
      ["#"] = true,
   },
   [2] = {
      ["."] = true,
      [":"] = true,
   },
}

local spaced_op = {
   [1] = {
      ["not"] = true,
   },
   [2] = {
      ["or"] = true,
      ["and"] = true,
      ["<"] = true,
      [">"] = true,
      ["<="] = true,
      [">="] = true,
      ["~="] = true,
      ["=="] = true,
      ["|"] = true,
      ["~"] = true,
      ["&"] = true,
      ["<<"] = true,
      [">>"] = true,
      [".."] = true,
      ["+"] = true,
      ["-"] = true,
      ["*"] = true,
      ["/"] = true,
      ["//"] = true,
      ["%"] = true,
      ["^"] = true,
   },
}

local PrettyPrintOpts = {}

local default_pretty_print_ast_opts = {
   preserve_indent = true,
   preserve_newlines = true,
}

local fast_pretty_print_ast_opts = {
   preserve_indent = false,
   preserve_newlines = true,
}

local primitive = {
   ["function"] = "function",
   ["enum"] = "string",
   ["boolean"] = "boolean",
   ["string"] = "string",
   ["nil"] = "nil",
   ["number"] = "number",
   ["integer"] = "number",
   ["thread"] = "thread",
}

function tl.pretty_print_ast(ast, mode)
   local indent = 0

   local opts
   if type(mode) == "table" then
      opts = mode
   elseif mode == true then
      opts = fast_pretty_print_ast_opts
   else
      opts = default_pretty_print_ast_opts
   end

   local Output = {}

   local save_indent = {}

   local function increment_indent(node)
      local child = node.body or node[1]
      if not child then
         return
      end
      if child.y ~= node.y then
         if indent == 0 and #save_indent > 0 then
            indent = save_indent[#save_indent] + 1
         else
            indent = indent + 1
         end
      else
         table.insert(save_indent, indent)
         indent = 0
      end
   end

   local function decrement_indent(node, child)
      if child.y ~= node.y then
         indent = indent - 1
      else
         indent = table.remove(save_indent)
      end
   end

   if not opts.preserve_indent then
      increment_indent = nil
      decrement_indent = function() end
   end

   local function add_string(out, s)
      table.insert(out, s)
      if string.find(s, "\n", 1, true) then
         for _nl in s:gmatch("\n") do
            out.h = out.h + 1
         end
      end
   end

   local function add_child(out, child, space, current_indent)
      if #child == 0 then
         return
      end

      if child.y ~= -1 and child.y < out.y then
         out.y = child.y
      end

      if child.y > out.y + out.h and opts.preserve_newlines then
         local delta = child.y - (out.y + out.h)
         out.h = out.h + delta
         table.insert(out, ("\n"):rep(delta))
      else
         if space then
            if space ~= "" then
               table.insert(out, space)
            end
            current_indent = nil
         end
      end
      if current_indent and opts.preserve_indent then
         table.insert(out, ("   "):rep(current_indent))
      end
      table.insert(out, child)
      out.h = out.h + child.h
   end

   local function concat_output(out)
      for i, s in ipairs(out) do
         if type(s) == "table" then
            out[i] = concat_output(s)
         end
      end
      return table.concat(out)
   end

   local function print_record_def(typ)
      local out = { "{" }
      for _, name in ipairs(typ.field_order) do
         if is_typetype(typ.fields[name]) and is_record_type(typ.fields[name].def) then
            table.insert(out, name)
            table.insert(out, " = ")
            table.insert(out, print_record_def(typ.fields[name].def))
            table.insert(out, ", ")
         end
      end
      table.insert(out, "}")
      return table.concat(out)
   end

   local visit_node = {}

   visit_node.cbs = {
      ["statements"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            local space
            for i, child in ipairs(children) do
               add_child(out, child, space, indent)
               if node[i].semicolon then
                  table.insert(out, ";")
                  space = " "
               else
                  space = "; "
               end
            end
            return out
         end,
      },
      ["local_declaration"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "local")
            add_child(out, children[1], " ")
            if children[3] then
               table.insert(out, " =")
               add_child(out, children[3], " ")
            end
            return out
         end,
      },
      ["local_type"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "local")
            add_child(out, children[1], " ")
            table.insert(out, " =")
            add_child(out, children[2], " ")
            return out
         end,
      },
      ["global_type"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            add_child(out, children[1], " ")
            table.insert(out, " =")
            add_child(out, children[2], " ")
            return out
         end,
      },
      ["global_declaration"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            if children[3] then
               add_child(out, children[1])
               table.insert(out, " =")
               add_child(out, children[3], " ")
            end
            return out
         end,
      },
      ["assignment"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            add_child(out, children[1])
            table.insert(out, " =")
            add_child(out, children[3], " ")
            return out
         end,
      },
      ["if"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            for i, child in ipairs(children) do
               add_child(out, child, i > 1 and " ", child.y ~= node.y and indent)
            end
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["if_block"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            if node.if_block_n == 1 then
               table.insert(out, "if")
            elseif not node.exp then
               table.insert(out, "else")
            else
               table.insert(out, "elseif")
            end
            if node.exp then
               add_child(out, children[1], " ")
               table.insert(out, " then")
            end
            add_child(out, children[2], " ")
            decrement_indent(node, node.body)
            return out
         end,
      },
      ["while"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "while")
            add_child(out, children[1], " ")
            table.insert(out, " do")
            add_child(out, children[2], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["repeat"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "repeat")
            add_child(out, children[1], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "until " }, " ", indent)
            add_child(out, children[2])
            return out
         end,
      },
      ["do"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "do")
            add_child(out, children[1], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["forin"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "for")
            add_child(out, children[1], " ")
            table.insert(out, " in")
            add_child(out, children[2], " ")
            table.insert(out, " do")
            add_child(out, children[3], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["fornum"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "for")
            add_child(out, children[1], " ")
            table.insert(out, " =")
            add_child(out, children[2], " ")
            table.insert(out, ",")
            add_child(out, children[3], " ")
            if children[4] then
               table.insert(out, ",")
               add_child(out, children[4], " ")
            end
            table.insert(out, " do")
            add_child(out, children[5], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["return"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "return")
            if #children[1] > 0 then
               add_child(out, children[1], " ")
            end
            return out
         end,
      },
      ["break"] = {
         after = function(node, _children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "break")
            return out
         end,
      },
      ["variable_list"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            local space
            for i, child in ipairs(children) do
               if i > 1 then
                  table.insert(out, ",")
                  space = " "
               end
               add_child(out, child, space, child.y ~= node.y and indent)
            end
            return out
         end,
      },
      ["table_literal"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            if #children == 0 then
               table.insert(out, "{}")
               return out
            end
            table.insert(out, "{")
            local n = #children
            for i, child in ipairs(children) do
               add_child(out, child, " ", child.y ~= node.y and indent)
               if i < n or node.yend ~= node.y then
                  table.insert(out, ",")
               end
            end
            decrement_indent(node, node[1])
            add_child(out, { y = node.yend, h = 0, [1] = "}" }, " ", indent)
            return out
         end,
      },
      ["table_item"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            if node.key_parsed ~= "implicit" then
               if node.key_parsed == "short" then
                  children[1][1] = children[1][1]:sub(2, -2)
                  add_child(out, children[1])
                  table.insert(out, " = ")
               else
                  table.insert(out, "[")
                  if node.key_parsed == "long" and node.key.is_longstring then
                     table.insert(children[1], 1, " ")
                     table.insert(children[1], " ")
                  end
                  add_child(out, children[1])
                  table.insert(out, "] = ")
               end
            end
            add_child(out, children[2])
            return out
         end,
      },
      ["local_function"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "local function")
            add_child(out, children[1], " ")
            table.insert(out, "(")
            add_child(out, children[2])
            table.insert(out, ")")
            add_child(out, children[4], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["global_function"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "function")
            add_child(out, children[1], " ")
            table.insert(out, "(")
            add_child(out, children[2])
            table.insert(out, ")")
            add_child(out, children[4], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["record_function"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "function")
            add_child(out, children[1], " ")
            table.insert(out, node.is_method and ":" or ".")
            add_child(out, children[2])
            table.insert(out, "(")
            if node.is_method then

               table.remove(children[3], 1)
               if children[3][1] == "," then
                  table.remove(children[3], 1)
                  table.remove(children[3], 1)
               end
            end
            add_child(out, children[3])
            table.insert(out, ")")
            add_child(out, children[5], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["function"] = {
         before = increment_indent,
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "function(")
            add_child(out, children[1])
            table.insert(out, ")")
            add_child(out, children[3], " ")
            decrement_indent(node, node.body)
            add_child(out, { y = node.yend, h = 0, [1] = "end" }, " ", indent)
            return out
         end,
      },
      ["cast"] = {},

      ["paren"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "(")
            add_child(out, children[1], "", indent)
            table.insert(out, ")")
            return out
         end,
      },
      ["op"] = {
         after = function(node, children)
            local out = { y = node.y, h = 0 }
            if node.op.op == "@funcall" then
               add_child(out, children[1], "", indent)
               table.insert(out, "(")
               add_child(out, children[3], "", indent)
               table.insert(out, ")")
            elseif node.op.op == "@index" then
               add_child(out, children[1], "", indent)
               table.insert(out, "[")
               if node.e2.is_longstring then
                  table.insert(children[3], 1, " ")
                  table.insert(children[3], " ")
               end
               add_child(out, children[3], "", indent)
               table.insert(out, "]")
            elseif node.op.op == "as" then
               add_child(out, children[1], "", indent)
            elseif node.op.op == "is" then
               if node.e2.casttype.typename == "integer" then
                  table.insert(out, "math.type(")
                  add_child(out, children[1], "", indent)
                  table.insert(out, ") == \"integer\"")
               else
                  table.insert(out, "type(")
                  add_child(out, children[1], "", indent)
                  table.insert(out, ") == \"")
                  add_child(out, children[3], "", indent)
                  table.insert(out, "\"")
               end
            elseif spaced_op[node.op.arity][node.op.op] or tight_op[node.op.arity][node.op.op] then
               local space = spaced_op[node.op.arity][node.op.op] and " " or ""
               if children[2] and node.op.prec > tonumber(children[2]) then
                  table.insert(children[1], 1, "(")
                  table.insert(children[1], ")")
               end
               if node.op.arity == 1 then
                  table.insert(out, node.op.op)
                  add_child(out, children[1], space, indent)
               elseif node.op.arity == 2 then
                  add_child(out, children[1], "", indent)
                  if space == " " then
                     table.insert(out, " ")
                  end
                  table.insert(out, node.op.op)
                  if children[4] and node.op.prec > tonumber(children[4]) then
                     table.insert(children[3], 1, "(")
                     table.insert(children[3], ")")
                  end
                  add_child(out, children[3], space, indent)
               end
            else
               error("unknown node op " .. node.op.op)
            end
            return out
         end,
      },
      ["variable"] = {
         after = function(node, _children)
            local out = { y = node.y, h = 0 }
            add_string(out, node.tk)
            return out
         end,
      },
      ["newtype"] = {
         after = function(node, _children)
            local out = { y = node.y, h = 0 }
            if node.is_alias then
               table.insert(out, table.concat(node.newtype.def.names, "."))
            elseif is_record_type(node.newtype.def) then
               table.insert(out, print_record_def(node.newtype.def))
            else
               table.insert(out, "{}")
            end
            return out
         end,
      },
      ["goto"] = {
         after = function(node, _children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "goto ")
            table.insert(out, node.label)
            return out
         end,
      },
      ["label"] = {
         after = function(node, _children)
            local out = { y = node.y, h = 0 }
            table.insert(out, "::")
            table.insert(out, node.label)
            table.insert(out, "::")
            return out
         end,
      },
   }

   local visit_type = {}
   visit_type.cbs = {
      ["string"] = {
         after = function(typ, _children)
            local out = { y = typ.y or -1, h = 0 }
            local r = typ.resolved or typ
            local lua_type = primitive[r.typename] or
            (r.is_userdata and "userdata") or
            "table"
            table.insert(out, lua_type)
            return out
         end,
      },
   }
   visit_type.cbs["typetype"] = visit_type.cbs["string"]
   visit_type.cbs["typevar"] = visit_type.cbs["string"]
   visit_type.cbs["typearg"] = visit_type.cbs["string"]
   visit_type.cbs["function"] = visit_type.cbs["string"]
   visit_type.cbs["thread"] = visit_type.cbs["string"]
   visit_type.cbs["array"] = visit_type.cbs["string"]
   visit_type.cbs["map"] = visit_type.cbs["string"]
   visit_type.cbs["tupletable"] = visit_type.cbs["string"]
   visit_type.cbs["arrayrecord"] = visit_type.cbs["string"]
   visit_type.cbs["record"] = visit_type.cbs["string"]
   visit_type.cbs["enum"] = visit_type.cbs["string"]
   visit_type.cbs["boolean"] = visit_type.cbs["string"]
   visit_type.cbs["nil"] = visit_type.cbs["string"]
   visit_type.cbs["number"] = visit_type.cbs["string"]
   visit_type.cbs["integer"] = visit_type.cbs["string"]
   visit_type.cbs["union"] = visit_type.cbs["string"]
   visit_type.cbs["nominal"] = visit_type.cbs["string"]
   visit_type.cbs["bad_nominal"] = visit_type.cbs["string"]
   visit_type.cbs["emptytable"] = visit_type.cbs["string"]
   visit_type.cbs["table_item"] = visit_type.cbs["string"]
   visit_type.cbs["unresolved_emptytable_value"] = visit_type.cbs["string"]
   visit_type.cbs["tuple"] = visit_type.cbs["string"]
   visit_type.cbs["poly"] = visit_type.cbs["string"]
   visit_type.cbs["any"] = visit_type.cbs["string"]
   visit_type.cbs["unknown"] = visit_type.cbs["string"]
   visit_type.cbs["invalid"] = visit_type.cbs["string"]
   visit_type.cbs["unresolved"] = visit_type.cbs["string"]
   visit_type.cbs["none"] = visit_type.cbs["string"]

   visit_node.cbs["expression_list"] = visit_node.cbs["variable_list"]
   visit_node.cbs["argument_list"] = visit_node.cbs["variable_list"]
   visit_node.cbs["identifier"] = visit_node.cbs["variable"]
   visit_node.cbs["number"] = visit_node.cbs["variable"]
   visit_node.cbs["integer"] = visit_node.cbs["variable"]
   visit_node.cbs["string"] = visit_node.cbs["variable"]
   visit_node.cbs["nil"] = visit_node.cbs["variable"]
   visit_node.cbs["boolean"] = visit_node.cbs["variable"]
   visit_node.cbs["..."] = visit_node.cbs["variable"]
   visit_node.cbs["argument"] = visit_node.cbs["variable"]
   visit_node.cbs["type_identifier"] = visit_node.cbs["variable"]

   local out = recurse_node(ast, visit_node, visit_type)
   local code
   if opts.preserve_newlines then
      code = { y = 1, h = 0 }
      add_child(code, out)
   else
      code = out
   end
   return concat_output(code)
end

local function VARARG(t)
   local tuple = t
   tuple.typename = "tuple"
   tuple.is_va = true
   return a_type(t)
end

local function TUPLE(t)
   local tuple = t
   tuple.typename = "tuple"
   return a_type(t)
end

local function UNION(t)
   return a_type({ typename = "union", types = t })
end

local NONE = a_type({ typename = "none" })
local INVALID = a_type({ typename = "invalid" })
local UNKNOWN = a_type({ typename = "unknown" })

local ALPHA = a_type({ typename = "typevar", typevar = "@a" })
local BETA = a_type({ typename = "typevar", typevar = "@b" })
local ARG_ALPHA = a_type({ typename = "typearg", typearg = "@a" })
local ARG_BETA = a_type({ typename = "typearg", typearg = "@b" })
local ARRAY_OF_ALPHA = a_type({ typename = "array", elements = ALPHA })
local MAP_OF_ALPHA_TO_BETA = a_type({ typename = "map", keys = ALPHA, values = BETA })
local NOMINAL_METATABLE_OF_ALPHA = a_type({ typename = "nominal", names = { "metatable" }, typevals = { ALPHA } })

local ARRAY_OF_STRING = a_type({ typename = "array", elements = STRING })

local FUNCTION = a_type({ typename = "function", args = VARARG({ ANY }), rets = VARARG({ ANY }) })

local NOMINAL_FILE = a_type({ typename = "nominal", names = { "FILE" } })
local XPCALL_MSGH_FUNCTION = a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({}) })

local USERDATA = ANY

local numeric_binop = {
   ["number"] = {
      ["number"] = NUMBER,
      ["integer"] = NUMBER,
   },
   ["integer"] = {
      ["integer"] = INTEGER,
      ["number"] = NUMBER,
   },
}

local float_binop = {
   ["number"] = {
      ["number"] = NUMBER,
      ["integer"] = NUMBER,
   },
   ["integer"] = {
      ["integer"] = NUMBER,
      ["number"] = NUMBER,
   },
}

local integer_binop = {
   ["number"] = {
      ["number"] = INTEGER,
      ["integer"] = INTEGER,
   },
   ["integer"] = {
      ["integer"] = INTEGER,
      ["number"] = INTEGER,
   },
}

local relational_binop = {
   ["number"] = {
      ["integer"] = BOOLEAN,
      ["number"] = BOOLEAN,
   },
   ["integer"] = {
      ["number"] = BOOLEAN,
      ["integer"] = BOOLEAN,
   },
   ["string"] = {
      ["string"] = BOOLEAN,
   },
   ["boolean"] = {
      ["boolean"] = BOOLEAN,
   },
}

local equality_binop = {
   ["number"] = {
      ["number"] = BOOLEAN,
      ["integer"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
   ["integer"] = {
      ["number"] = BOOLEAN,
      ["integer"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
   ["string"] = {
      ["string"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
   ["boolean"] = {
      ["boolean"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
   ["record"] = {
      ["emptytable"] = BOOLEAN,
      ["arrayrecord"] = BOOLEAN,
      ["record"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
   ["array"] = {
      ["emptytable"] = BOOLEAN,
      ["arrayrecord"] = BOOLEAN,
      ["array"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
   ["arrayrecord"] = {
      ["emptytable"] = BOOLEAN,
      ["arrayrecord"] = BOOLEAN,
      ["record"] = BOOLEAN,
      ["array"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
   ["map"] = {
      ["emptytable"] = BOOLEAN,
      ["map"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
   ["thread"] = {
      ["thread"] = BOOLEAN,
      ["nil"] = BOOLEAN,
   },
}

local unop_types = {
   ["#"] = {
      ["arrayrecord"] = INTEGER,
      ["string"] = INTEGER,
      ["array"] = INTEGER,
      ["tupletable"] = INTEGER,
      ["map"] = INTEGER,
      ["emptytable"] = INTEGER,
   },
   ["-"] = {
      ["number"] = NUMBER,
      ["integer"] = INTEGER,
   },
   ["~"] = {
      ["number"] = INTEGER,
      ["integer"] = INTEGER,
   },
   ["not"] = {
      ["string"] = BOOLEAN,
      ["number"] = BOOLEAN,
      ["integer"] = BOOLEAN,
      ["boolean"] = BOOLEAN,
      ["record"] = BOOLEAN,
      ["arrayrecord"] = BOOLEAN,
      ["array"] = BOOLEAN,
      ["tupletable"] = BOOLEAN,
      ["map"] = BOOLEAN,
      ["emptytable"] = BOOLEAN,
      ["thread"] = BOOLEAN,
   },
}

local unop_to_metamethod = {
   ["#"] = "__len",
   ["-"] = "__unm",
   ["~"] = "__bnot",
}

local binop_types = {
   ["+"] = numeric_binop,
   ["-"] = numeric_binop,
   ["*"] = numeric_binop,
   ["%"] = numeric_binop,
   ["/"] = float_binop,
   ["//"] = numeric_binop,
   ["^"] = float_binop,
   ["&"] = integer_binop,
   ["|"] = integer_binop,
   ["<<"] = integer_binop,
   [">>"] = integer_binop,
   ["~"] = integer_binop,
   ["=="] = equality_binop,
   ["~="] = equality_binop,
   ["<="] = relational_binop,
   [">="] = relational_binop,
   ["<"] = relational_binop,
   [">"] = relational_binop,
   ["or"] = {
      ["boolean"] = {
         ["boolean"] = BOOLEAN,
         ["function"] = FUNCTION,
      },
      ["number"] = {
         ["integer"] = NUMBER,
         ["number"] = NUMBER,
         ["boolean"] = BOOLEAN,
      },
      ["integer"] = {
         ["integer"] = INTEGER,
         ["number"] = NUMBER,
         ["boolean"] = BOOLEAN,
      },
      ["string"] = {
         ["string"] = STRING,
         ["boolean"] = BOOLEAN,
         ["enum"] = STRING,
      },
      ["function"] = {
         ["boolean"] = BOOLEAN,
      },
      ["array"] = {
         ["boolean"] = BOOLEAN,
      },
      ["record"] = {
         ["boolean"] = BOOLEAN,
      },
      ["arrayrecord"] = {
         ["boolean"] = BOOLEAN,
      },
      ["map"] = {
         ["boolean"] = BOOLEAN,
      },
      ["enum"] = {
         ["string"] = STRING,
      },
      ["thread"] = {
         ["boolean"] = BOOLEAN,
      },
   },
   [".."] = {
      ["string"] = {
         ["string"] = STRING,
         ["enum"] = STRING,
         ["number"] = STRING,
         ["integer"] = STRING,
      },
      ["number"] = {
         ["integer"] = STRING,
         ["number"] = STRING,
         ["string"] = STRING,
         ["enum"] = STRING,
      },
      ["integer"] = {
         ["integer"] = STRING,
         ["number"] = STRING,
         ["string"] = STRING,
         ["enum"] = STRING,
      },
      ["enum"] = {
         ["number"] = STRING,
         ["integer"] = STRING,
         ["string"] = STRING,
         ["enum"] = STRING,
      },
   },
}

local binop_to_metamethod = {
   ["+"] = "__add",
   ["-"] = "__sub",
   ["*"] = "__mul",
   ["/"] = "__div",
   ["%"] = "__mod",
   ["^"] = "__pow",
   ["//"] = "__idiv",
   ["&"] = "__band",
   ["|"] = "__bor",
   ["~"] = "__bxor",
   ["<<"] = "__shl",
   [">>"] = "__shr",
   [".."] = "__concat",
   ["=="] = "__eq",
   ["<"] = "__lt",
   ["<="] = "__le",
}

local function is_unknown(t)
   return t.typename == "unknown" or
   t.typename == "unresolved_emptytable_value"
end

local show_type

local function show_type_base(t, short, seen)

   if seen[t] then
      return seen[t]
   end
   seen[t] = "..."

   local function show(typ)
      return show_type(typ, short, seen)
   end

   if t.typename == "nominal" then
      if t.typevals then
         local out = { table.concat(t.names, "."), "<" }
         local vals = {}
         for _, v in ipairs(t.typevals) do
            table.insert(vals, show(v))
         end
         table.insert(out, table.concat(vals, ", "))
         table.insert(out, ">")
         return table.concat(out)
      else
         return table.concat(t.names, ".")
      end
   elseif t.typename == "tuple" then
      local out = {}
      for _, v in ipairs(t) do
         table.insert(out, show(v))
      end
      return "(" .. table.concat(out, ", ") .. ")"
   elseif t.typename == "tupletable" then
      local out = {}
      for _, v in ipairs(t.types) do
         table.insert(out, show(v))
      end
      return "{" .. table.concat(out, ", ") .. "}"
   elseif t.typename == "poly" then
      local out = {}
      for _, v in ipairs(t.types) do
         table.insert(out, show(v))
      end
      return table.concat(out, " and ")
   elseif t.typename == "union" then
      local out = {}
      for _, v in ipairs(t.types) do
         table.insert(out, show(v))
      end
      return table.concat(out, " | ")
   elseif t.typename == "emptytable" then
      return "{}"
   elseif t.typename == "map" then
      return "{" .. show(t.keys) .. " : " .. show(t.values) .. "}"
   elseif t.typename == "array" then
      return "{" .. show(t.elements) .. "}"
   elseif t.typename == "enum" then
      return t.names and table.concat(t.names, ".") or "enum"
   elseif is_record_type(t) then
      if short then
         return "record"
      else
         local out = { "record" }
         if t.typeargs then
            table.insert(out, "<")
            local typeargs = {}
            for _, v in ipairs(t.typeargs) do
               table.insert(typeargs, show(v))
            end
            table.insert(out, table.concat(typeargs, ", "))
            table.insert(out, ">")
         end
         table.insert(out, " (")
         if t.elements then
            table.insert(out, "{" .. show(t.elements) .. "}")
         end
         local fs = {}
         for _, k in ipairs(t.field_order) do
            local v = t.fields[k]
            table.insert(fs, k .. ": " .. show(v))
         end
         table.insert(out, table.concat(fs, "; "))
         table.insert(out, ")")
         return table.concat(out)
      end
   elseif t.typename == "function" then
      local out = { "function" }
      if t.typeargs then
         table.insert(out, "<")
         local typeargs = {}
         for _, v in ipairs(t.typeargs) do
            table.insert(typeargs, show(v))
         end
         table.insert(out, table.concat(typeargs, ", "))
         table.insert(out, ">")
      end
      table.insert(out, "(")
      local args = {}
      if t.is_method then
         table.insert(args, "self")
      end
      for i, v in ipairs(t.args) do
         if not t.is_method or i > 1 then
            table.insert(args, (i == #t.args and t.args.is_va and "...: " or "") .. show(v))
         end
      end
      table.insert(out, table.concat(args, ", "))
      table.insert(out, ")")
      if #t.rets > 0 then
         table.insert(out, ": ")
         local rets = {}
         for i, v in ipairs(t.rets) do
            table.insert(rets, show(v) .. (i == #t.rets and t.rets.is_va and "..." or ""))
         end
         table.insert(out, table.concat(rets, ", "))
      end
      return table.concat(out)
   elseif t.typename == "number" or
      t.typename == "integer" or
      t.typename == "boolean" or
      t.typename == "thread" then
      return t.typename
   elseif t.typename == "string" then
      if short then
         return "string"
      else
         return t.typename ..
         (t.tk and " " .. t.tk or "")
      end
   elseif t.typename == "typevar" then
      return t.typevar
   elseif t.typename == "typearg" then
      return t.typearg
   elseif is_unknown(t) then
      return "<unknown type>"
   elseif t.typename == "invalid" then
      return "<invalid type>"
   elseif t.typename == "any" then
      return "<any type>"
   elseif t.typename == "nil" then
      return "nil"
   elseif t.typename == "none" then
      return ""
   elseif is_typetype(t) then
      return "type " .. show(t.def)
   elseif t.typename == "bad_nominal" then
      return table.concat(t.names, ".") .. " (an unknown type)"
   else
      return tostring(t)
   end
end

local function inferred_msg(t)
   return " (inferred at " .. t.inferred_at_file .. ":" .. t.inferred_at.y .. ":" .. t.inferred_at.x .. ")"
end

show_type = function(t, short, seen)
   seen = seen or {}
   local ret = show_type_base(t, short, seen)
   if t.inferred_at then
      ret = ret .. inferred_msg(t)
   end
   seen[t] = ret
   return ret
end

local function search_for(module_name, suffix, path, tried)
   for entry in path:gmatch("[^;]+") do
      local slash_name = module_name:gsub("%.", "/")
      local filename = entry:gsub("?", slash_name)
      local tl_filename = filename:gsub("%.lua$", suffix)
      local fd = io.open(tl_filename, "r")
      if fd then
         return tl_filename, fd, tried
      end
      table.insert(tried, "no file '" .. tl_filename .. "'")
   end
   return nil, nil, tried
end

function tl.search_module(module_name, search_dtl)
   local found
   local fd
   local tried = {}
   local path = os.getenv("TL_PATH") or package.path
   if search_dtl then
      found, fd, tried = search_for(module_name, ".d.tl", path, tried)
      if found then
         return found, fd
      end
   end
   found, fd, tried = search_for(module_name, ".tl", path, tried)
   if found then
      return found, fd
   end
   found, fd, tried = search_for(module_name, ".lua", path, tried)
   if found then
      return found, fd
   end
   return nil, nil, tried
end

local Variable = {}

local function sorted_keys(m)
   local keys = {}
   for k, _ in pairs(m) do
      table.insert(keys, k)
   end
   table.sort(keys)
   return keys
end

local function fill_field_order(t)
   if t.typename == "record" then
      t.field_order = sorted_keys(t.fields)
   end
end

local function require_module(module_name, lax, env)
   local modules = env.modules

   if modules[module_name] then
      return modules[module_name], true
   end
   modules[module_name] = INVALID

   local found, fd = tl.search_module(module_name, true)
   if found and (lax or found:match("tl$")) then
      fd:close()
      local found_result, err = tl.process(found, env)
      assert(found_result, err)

      if not found_result.type then
         found_result.type = BOOLEAN
      end

      env.modules[module_name] = found_result.type

      return found_result.type, true
   end

   return INVALID, found ~= nil
end

local compat_code_cache = {}

local function add_compat_entries(program, used_set, gen_compat)
   if gen_compat == "off" or not next(used_set) then
      return
   end

   local used_list = sorted_keys(used_set)

   local compat_loaded = false

   local n = 1
   local function load_code(name, text)
      local code = compat_code_cache[name]
      if not code then
         local tokens = tl.lex(text)
         local _
         _, code = tl.parse_program(tokens, {}, "@internal")
         tl.type_check(code, { filename = "<internal>", lax = false, gen_compat = "off" })
         code = code
         compat_code_cache[name] = code
      end
      for _, c in ipairs(code) do
         table.insert(program, n, c)
         n = n + 1
      end
   end

   local function req(m)
      return (gen_compat == "optional") and
      "pcall(require, '" .. m .. "')" or
      "true, require('" .. m .. "')"
   end

   for _, name in ipairs(used_list) do
      if name == "table.unpack" then
         load_code(name, "local _tl_table_unpack = unpack or table.unpack")
      elseif name == "bit32" then
         load_code(name, "local bit32 = bit32; if not bit32 then local p, m = " .. req("bit32") .. "; if p then bit32 = m end")
      elseif name == "mt" then
         load_code(name, "local _tl_mt = function(m, s, a, b) return (getmetatable(s == 1 and a or b)[m](a, b) end")
      else
         if not compat_loaded then
            load_code("compat", "local _tl_compat; if (tonumber((_VERSION or ''):match('[%d.]*$')) or 0) < 5.3 then local p, m = " .. req("compat53.module") .. "; if p then _tl_compat = m end")
            compat_loaded = true
         end
         load_code(name, (("local $NAME = _tl_compat and _tl_compat.$NAME or $NAME"):gsub("$NAME", name)))
      end
   end
   program.y = 1
end

local function get_stdlib_compat(lax)
   if lax then
      return {
         ["utf8"] = true,
      }
   else
      return {
         ["io"] = true,
         ["math"] = true,
         ["string"] = true,
         ["table"] = true,
         ["utf8"] = true,
         ["coroutine"] = true,
         ["os"] = true,
         ["package"] = true,
         ["debug"] = true,
         ["load"] = true,
         ["loadfile"] = true,
         ["assert"] = true,
         ["pairs"] = true,
         ["ipairs"] = true,
         ["pcall"] = true,
         ["xpcall"] = true,
         ["rawlen"] = true,
      }
   end
end

local bit_operators = {
   ["&"] = "band",
   ["|"] = "bor",
   ["~"] = "bxor",
   [">>"] = "rshift",
   ["<<"] = "lshift",
}

local function convert_node_to_compat_call(node, mod_name, fn_name, e1, e2)
   node.op.op = "@funcall"
   node.op.arity = 2
   node.op.prec = 100
   node.e1 = { y = node.y, x = node.x, kind = "op", op = an_operator(node, 2, ".") }
   node.e1.e1 = { y = node.y, x = node.x, kind = "identifier", tk = mod_name }
   node.e1.e2 = { y = node.y, x = node.x, kind = "identifier", tk = fn_name }
   node.e2 = { y = node.y, x = node.x, kind = "expression_list" }
   node.e2[1] = e1
   node.e2[2] = e2
end

local function convert_node_to_compat_mt_call(node, mt_name, which_self, e1, e2)
   node.op.op = "@funcall"
   node.op.arity = 2
   node.op.prec = 100
   node.e1 = { y = node.y, x = node.x, kind = "identifier", tk = "_tl_mt" }
   node.e2 = { y = node.y, x = node.x, kind = "expression_list" }
   node.e2[1] = { y = node.y, x = node.x, kind = "string", tk = "\"" .. mt_name .. "\"" }
   node.e2[2] = { y = node.y, x = node.x, kind = "integer", tk = tostring(which_self) }
   node.e2[3] = e1
   node.e2[4] = e2
end

local globals_typeid

local function init_globals(lax)
   local globals = {}
   local stdlib_compat = get_stdlib_compat(lax)


   local is_first_init = globals_typeid == nil

   local save_typeid = last_typeid
   if is_first_init then
      globals_typeid = last_typeid
   else
      last_typeid = globals_typeid
   end

   local LOAD_FUNCTION = a_type({ typename = "function", args = {}, rets = TUPLE({ STRING }) })

   local OS_DATE_TABLE = a_type({
      typename = "record",
      fields = {
         ["year"] = INTEGER,
         ["month"] = INTEGER,
         ["day"] = INTEGER,
         ["hour"] = INTEGER,
         ["min"] = INTEGER,
         ["sec"] = INTEGER,
         ["wday"] = INTEGER,
         ["yday"] = INTEGER,
         ["isdst"] = BOOLEAN,
      },
   })

   local OS_DATE_TABLE_FORMAT = a_type({ typename = "enum", enumset = { ["!*t"] = true, ["*t"] = true } })

   local DEBUG_GETINFO_TABLE = a_type({
      typename = "record",
      fields = {
         ["name"] = STRING,
         ["namewhat"] = STRING,
         ["source"] = STRING,
         ["short_src"] = STRING,
         ["linedefined"] = INTEGER,
         ["lastlinedefined"] = INTEGER,
         ["what"] = STRING,
         ["currentline"] = INTEGER,
         ["istailcall"] = BOOLEAN,
         ["nups"] = INTEGER,
         ["nparams"] = INTEGER,
         ["isvararg"] = BOOLEAN,
         ["func"] = ANY,
         ["activelines"] = a_type({ typename = "map", keys = INTEGER, values = BOOLEAN }),
      },
   })

   local DEBUG_HOOK_EVENT = a_type({
      typename = "enum",
      enumset = {
         ["call"] = true,
         ["tail call"] = true,
         ["return"] = true,
         ["line"] = true,
         ["count"] = true,
      },
   })
   local DEBUG_HOOK_FUNCTION = a_type({
      typename = "function",
      args = TUPLE({ DEBUG_HOOK_EVENT, INTEGER }),
      rets = TUPLE({}),
   })

   local TABLE_SORT_FUNCTION = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ALPHA, ALPHA }), rets = TUPLE({ BOOLEAN }) })

   local OPT_NUMBER = NUMBER
   local OPT_STRING = STRING
   local OPT_THREAD = THREAD
   local OPT_ALPHA = ALPHA
   local OPT_BETA = BETA
   local OPT_TABLE = TABLE
   local OPT_UNION = UNION
   local OPT_BOOLEAN = BOOLEAN
   local OPT_NOMINAL_FILE = NOMINAL_FILE
   local OPT_TABLE_SORT_FUNCTION = TABLE_SORT_FUNCTION

   local standard_library = {
      ["..."] = VARARG({ STRING }),
      ["any"] = a_type({ typename = "typetype", def = ANY }),
      ["arg"] = ARRAY_OF_STRING,
      ["assert"] = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA, ARG_BETA }), args = TUPLE({ ALPHA, OPT_BETA }), rets = TUPLE({ ALPHA }) }),
      ["collectgarbage"] = a_type({
         typename = "poly",
         types = {
            a_type({ typename = "function", args = TUPLE({ a_type({ typename = "enum", enumset = { ["collect"] = true, ["count"] = true, ["stop"] = true, ["restart"] = true } }) }), rets = TUPLE({ NUMBER }) }),
            a_type({ typename = "function", args = TUPLE({ a_type({ typename = "enum", enumset = { ["step"] = true, ["setpause"] = true, ["setstepmul"] = true } }), NUMBER }), rets = TUPLE({ NUMBER }) }),
            a_type({ typename = "function", args = TUPLE({ a_type({ typename = "enum", enumset = { ["isrunning"] = true } }) }), rets = TUPLE({ BOOLEAN }) }),
            a_type({ typename = "function", args = TUPLE({ STRING, OPT_NUMBER }), rets = TUPLE({ a_type({ typename = "union", types = { BOOLEAN, NUMBER } }) }) }),
         },
      }),
      ["dofile"] = a_type({ typename = "function", args = TUPLE({ OPT_STRING }), rets = VARARG({ ANY }) }),
      ["error"] = a_type({ typename = "function", args = TUPLE({ ANY, NUMBER }), rets = TUPLE({}) }),
      ["getmetatable"] = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ALPHA }), rets = TUPLE({ NOMINAL_METATABLE_OF_ALPHA }) }),
      ["ipairs"] = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ARRAY_OF_ALPHA }), rets = TUPLE({
         a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ INTEGER, ALPHA }) }),
      }), }),
      ["load"] = a_type({ typename = "function", args = TUPLE({ UNION({ STRING, LOAD_FUNCTION }), OPT_STRING, OPT_STRING, OPT_TABLE }), rets = TUPLE({ FUNCTION, STRING }) }),
      ["loadfile"] = a_type({ typename = "function", args = TUPLE({ OPT_STRING, OPT_STRING, OPT_TABLE }), rets = TUPLE({ FUNCTION, STRING }) }),
      ["next"] = a_type({
         typename = "poly",
         types = {
            a_type({ typeargs = TUPLE({ ARG_ALPHA, ARG_BETA }), typename = "function", args = TUPLE({ MAP_OF_ALPHA_TO_BETA, OPT_ALPHA }), rets = TUPLE({ ALPHA, BETA }) }),
            a_type({ typeargs = TUPLE({ ARG_ALPHA }), typename = "function", args = TUPLE({ ARRAY_OF_ALPHA, OPT_ALPHA }), rets = TUPLE({ INTEGER, ALPHA }) }),
         },
      }),
      ["pairs"] = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA, ARG_BETA }), args = TUPLE({ a_type({ typename = "map", keys = ALPHA, values = BETA }) }), rets = TUPLE({
         a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ ALPHA, BETA }) }),
      }), }),
      ["pcall"] = a_type({ typename = "function", args = VARARG({ FUNCTION, ANY }), rets = TUPLE({ BOOLEAN, ANY }) }),
      ["xpcall"] = a_type({ typename = "function", args = VARARG({ FUNCTION, XPCALL_MSGH_FUNCTION, ANY }), rets = TUPLE({ BOOLEAN, ANY }) }),
      ["print"] = a_type({ typename = "function", args = VARARG({ ANY }), rets = TUPLE({}) }),
      ["rawequal"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ BOOLEAN }) }),
      ["rawget"] = a_type({ typename = "function", args = TUPLE({ TABLE, ANY }), rets = TUPLE({ ANY }) }),
      ["rawlen"] = a_type({ typename = "function", args = TUPLE({ UNION({ TABLE, STRING }) }), rets = TUPLE({ INTEGER }) }),
      ["rawset"] = a_type({
         typename = "poly",
         types = {
            a_type({ typeargs = TUPLE({ ARG_ALPHA, ARG_BETA }), typename = "function", args = TUPLE({ MAP_OF_ALPHA_TO_BETA, ALPHA, BETA }), rets = TUPLE({}) }),
            a_type({ typeargs = TUPLE({ ARG_ALPHA }), typename = "function", args = TUPLE({ ARRAY_OF_ALPHA, NUMBER, ALPHA }), rets = TUPLE({}) }),
            a_type({ typename = "function", args = TUPLE({ TABLE, ANY, ANY }), rets = TUPLE({}) }),
         },
      }),
      ["require"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({}) }),
      ["select"] = a_type({
         typename = "poly",
         types = {
            a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = VARARG({ NUMBER, ALPHA }), rets = TUPLE({ ALPHA }) }),
            a_type({ typename = "function", args = VARARG({ NUMBER, ANY }), rets = TUPLE({ ANY }) }),
            a_type({ typename = "function", args = VARARG({ STRING, ANY }), rets = TUPLE({ INTEGER }) }),
         },
      }),
      ["setmetatable"] = a_type({ typeargs = TUPLE({ ARG_ALPHA }), typename = "function", args = TUPLE({ ALPHA, NOMINAL_METATABLE_OF_ALPHA }), rets = TUPLE({ ALPHA }) }),
      ["tonumber"] = a_type({
         typename = "poly",
         types = {
            a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ NUMBER }) }),
            a_type({ typename = "function", args = TUPLE({ ANY, NUMBER }), rets = TUPLE({ INTEGER }) }),
         },
      }),
      ["tostring"] = a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ STRING }) }),
      ["type"] = a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ STRING }) }),
      ["FILE"] = a_type({
         typename = "typetype",
         def = a_type({
            typename = "record",
            is_userdata = true,
            fields = {
               ["close"] = a_type({ typename = "function", args = TUPLE({ NOMINAL_FILE }), rets = TUPLE({ BOOLEAN, STRING }) }),
               ["flush"] = a_type({ typename = "function", args = TUPLE({ NOMINAL_FILE }), rets = TUPLE({}) }),
               ["lines"] = a_type({ typename = "function", args = VARARG({ NOMINAL_FILE, a_type({ typename = "union", types = { STRING, NUMBER } }) }), rets = TUPLE({
                  a_type({ typename = "function", args = TUPLE({}), rets = VARARG({ STRING }) }),
               }), }),
               ["read"] = a_type({ typename = "function", args = TUPLE({ NOMINAL_FILE, UNION({ STRING, NUMBER }) }), rets = TUPLE({ STRING, STRING }) }),
               ["seek"] = a_type({ typename = "function", args = TUPLE({ NOMINAL_FILE, OPT_STRING, OPT_NUMBER }), rets = TUPLE({ INTEGER, STRING }) }),
               ["setvbuf"] = a_type({ typename = "function", args = TUPLE({ NOMINAL_FILE, STRING, OPT_NUMBER }), rets = TUPLE({}) }),
               ["write"] = a_type({ typename = "function", args = VARARG({ NOMINAL_FILE, STRING }), rets = TUPLE({ NOMINAL_FILE, STRING }) }),

            },
         }),
      }),
      ["metatable"] = a_type({
         typename = "typetype",
         def = a_type({
            typename = "record",
            typeargs = TUPLE({ ARG_ALPHA }),
            fields = {
               ["__call"] = a_type({ typename = "function", args = VARARG({ ALPHA, ANY }), rets = VARARG({ ANY }) }),
               ["__gc"] = a_type({ typename = "function", args = TUPLE({ ALPHA }), rets = TUPLE({}) }),
               ["__index"] = ANY,
               ["__len"] = a_type({ typename = "function", args = TUPLE({ ALPHA }), rets = TUPLE({ ANY }) }),
               ["__mode"] = a_type({ typename = "enum", enumset = { ["k"] = true, ["v"] = true, ["kv"] = true } }),
               ["__newindex"] = ANY,
               ["__pairs"] = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA, ARG_BETA }),
args = TUPLE({ a_type({ typename = "map", keys = ALPHA, values = BETA }) }),
rets = TUPLE({ a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ ALPHA, BETA }) }) }),
               }),
               ["__tostring"] = a_type({ typename = "function", args = TUPLE({ ALPHA }), rets = TUPLE({ STRING }) }),
               ["__name"] = STRING,
               ["__add"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__sub"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__mul"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__div"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__idiv"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__mod"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__pow"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__unm"] = a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ ANY }) }),
               ["__band"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__bor"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__bxor"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__bnot"] = a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ ANY }) }),
               ["__shl"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__shr"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__concat"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ ANY }) }),
               ["__eq"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ BOOLEAN }) }),
               ["__lt"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ BOOLEAN }) }),
               ["__le"] = a_type({ typename = "function", args = TUPLE({ ANY, ANY }), rets = TUPLE({ BOOLEAN }) }),
            },
         }),
      }),
      ["coroutine"] = a_type({
         typename = "record",
         fields = {
            ["create"] = a_type({ typename = "function", args = TUPLE({ FUNCTION }), rets = TUPLE({ THREAD }) }),
            ["close"] = a_type({ typename = "function", args = TUPLE({ THREAD }), rets = TUPLE({ BOOLEAN, STRING }) }),
            ["isyieldable"] = a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ BOOLEAN }) }),
            ["resume"] = a_type({ typename = "function", args = VARARG({ THREAD, ANY }), rets = VARARG({ BOOLEAN, ANY }) }),
            ["running"] = a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ THREAD, BOOLEAN }) }),
            ["status"] = a_type({ typename = "function", args = TUPLE({ THREAD }), rets = TUPLE({ STRING }) }),
            ["wrap"] = a_type({ typename = "function", args = TUPLE({ FUNCTION }), rets = TUPLE({ FUNCTION }) }),
            ["yield"] = a_type({ typename = "function", args = VARARG({ ANY }), rets = VARARG({ ANY }) }),
         },
      }),
      ["debug"] = a_type({
         typename = "record",
         fields = {
            ["Info"] = a_type({
               typename = "typetype",
               def = DEBUG_GETINFO_TABLE,
            }),
            ["Hook"] = a_type({
               typename = "typetype",
               def = DEBUG_HOOK_FUNCTION,
            }),
            ["HookEvent"] = a_type({
               typename = "typetype",
               def = DEBUG_HOOK_EVENT,
            }),

            ["debug"] = a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({}) }),
            ["gethook"] = a_type({ typename = "function", args = TUPLE({ OPT_THREAD }), rets = TUPLE({ DEBUG_HOOK_FUNCTION, INTEGER }) }),
            ["getlocal"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ THREAD, FUNCTION, NUMBER }), rets = TUPLE({}) }),
                  a_type({ typename = "function", args = TUPLE({ FUNCTION, NUMBER }), rets = TUPLE({}) }),
               },
            }),
            ["getmetatable"] = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ALPHA }), rets = TUPLE({ NOMINAL_METATABLE_OF_ALPHA }) }),
            ["getregistry"] = a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ TABLE }) }),
            ["getupvalue"] = a_type({ typename = "function", args = TUPLE({ FUNCTION, NUMBER }), rets = TUPLE({ ANY }) }),
            ["getuservalue"] = a_type({ typename = "function", args = TUPLE({ USERDATA, NUMBER }), rets = TUPLE({ ANY }) }),
            ["sethook"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ THREAD, DEBUG_HOOK_FUNCTION, STRING, NUMBER }), rets = TUPLE({}) }),
                  a_type({ typename = "function", args = TUPLE({ DEBUG_HOOK_FUNCTION, STRING, NUMBER }), rets = TUPLE({}) }),
               },
            }),
            ["setlocal"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ THREAD, NUMBER, NUMBER, ANY }), rets = TUPLE({ STRING }) }),
                  a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER, ANY }), rets = TUPLE({ STRING }) }),
               },
            }),
            ["setmetatable"] = a_type({ typeargs = TUPLE({ ARG_ALPHA }), typename = "function", args = TUPLE({ ALPHA, NOMINAL_METATABLE_OF_ALPHA }), rets = TUPLE({ ALPHA }) }),
            ["setupvalue"] = a_type({ typename = "function", args = TUPLE({ FUNCTION, NUMBER, ANY }), rets = TUPLE({ STRING }) }),
            ["setuservalue"] = a_type({ typename = "function", args = TUPLE({ USERDATA, ANY, NUMBER }), rets = TUPLE({ USERDATA }) }),
            ["traceback"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ THREAD, STRING, NUMBER }), rets = TUPLE({ STRING }) }),
                  a_type({ typename = "function", args = TUPLE({ STRING, NUMBER }), rets = TUPLE({ STRING }) }),
               },
            }),
            ["upvalueid"] = a_type({ typename = "function", args = TUPLE({ FUNCTION, NUMBER }), rets = TUPLE({ USERDATA }) }),
            ["upvaluejoin"] = a_type({ typename = "function", args = TUPLE({ FUNCTION, NUMBER, FUNCTION, NUMBER }), rets = TUPLE({}) }),
            ["getinfo"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ DEBUG_GETINFO_TABLE }) }),
                  a_type({ typename = "function", args = TUPLE({ ANY, STRING }), rets = TUPLE({ DEBUG_GETINFO_TABLE }) }),
                  a_type({ typename = "function", args = TUPLE({ ANY, ANY, STRING }), rets = TUPLE({ DEBUG_GETINFO_TABLE }) }),
               },
            }),
         },
      }),
      ["io"] = a_type({
         typename = "record",
         fields = {
            ["close"] = a_type({ typename = "function", args = TUPLE({ OPT_NOMINAL_FILE }), rets = TUPLE({ BOOLEAN, STRING }) }),
            ["flush"] = a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({}) }),
            ["input"] = a_type({ typename = "function", args = TUPLE({ OPT_UNION({ STRING, NOMINAL_FILE }) }), rets = TUPLE({ NOMINAL_FILE }) }),
            ["lines"] = a_type({ typename = "function", args = VARARG({ OPT_STRING, a_type({ typename = "union", types = { STRING, NUMBER } }) }), rets = TUPLE({
               a_type({ typename = "function", args = TUPLE({}), rets = VARARG({ STRING }) }),
            }), }),
            ["open"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING }), rets = TUPLE({ NOMINAL_FILE, STRING }) }),
            ["output"] = a_type({ typename = "function", args = TUPLE({ OPT_UNION({ STRING, NOMINAL_FILE }) }), rets = TUPLE({ NOMINAL_FILE }) }),
            ["popen"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING }), rets = TUPLE({ NOMINAL_FILE, STRING }) }),
            ["read"] = a_type({ typename = "function", args = TUPLE({ UNION({ STRING, NUMBER }) }), rets = TUPLE({ STRING, STRING }) }),
            ["stderr"] = NOMINAL_FILE,
            ["stdin"] = NOMINAL_FILE,
            ["stdout"] = NOMINAL_FILE,
            ["tmpfile"] = a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ NOMINAL_FILE }) }),
            ["type"] = a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ STRING }) }),
            ["write"] = a_type({ typename = "function", args = VARARG({ STRING }), rets = TUPLE({ NOMINAL_FILE, STRING }) }),
         },
      }),
      ["math"] = a_type({
         typename = "record",
         fields = {
            ["abs"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ INTEGER }), rets = TUPLE({ INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
               },
            }),
            ["acos"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["asin"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["atan"] = a_type({ typename = "function", args = TUPLE({ NUMBER, OPT_NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["atan2"] = a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["ceil"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ INTEGER }) }),
            ["cos"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["cosh"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["deg"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["exp"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["floor"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ INTEGER }) }),
            ["fmod"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ INTEGER, INTEGER }), rets = TUPLE({ INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ NUMBER }) }),
               },
            }),
            ["frexp"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER, NUMBER }) }),
            ["huge"] = NUMBER,
            ["ldexp"] = a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["log"] = a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["log10"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["max"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = VARARG({ INTEGER }), rets = TUPLE({ INTEGER }) }),
                  a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = VARARG({ ALPHA }), rets = TUPLE({ ALPHA }) }),
                  a_type({ typename = "function", args = VARARG({ a_type({ typename = "union", types = { NUMBER, INTEGER } }) }), rets = TUPLE({ NUMBER }) }),
                  a_type({ typename = "function", args = VARARG({ ANY }), rets = TUPLE({ ANY }) }),
               },
            }),
            ["maxinteger"] = INTEGER,
            ["min"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = VARARG({ INTEGER }), rets = TUPLE({ INTEGER }) }),
                  a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = VARARG({ ALPHA }), rets = TUPLE({ ALPHA }) }),
                  a_type({ typename = "function", args = VARARG({ a_type({ typename = "union", types = { NUMBER, INTEGER } }) }), rets = TUPLE({ NUMBER }) }),
                  a_type({ typename = "function", args = VARARG({ ANY }), rets = TUPLE({ ANY }) }),
               },
            }),
            ["mininteger"] = INTEGER,
            ["modf"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ INTEGER, NUMBER }) }),
            ["pi"] = NUMBER,
            ["pow"] = a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["rad"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["random"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ NUMBER }) }),
               },
            }),
            ["randomseed"] = a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ INTEGER, INTEGER }) }),
            ["sin"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["sinh"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["sqrt"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["tan"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["tanh"] = a_type({ typename = "function", args = TUPLE({ NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["tointeger"] = a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ INTEGER }) }),
            ["type"] = a_type({ typename = "function", args = TUPLE({ ANY }), rets = TUPLE({ STRING }) }),
            ["ult"] = a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ BOOLEAN }) }),
         },
      }),
      ["os"] = a_type({
         typename = "record",
         fields = {
            ["clock"] = a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ NUMBER }) }),
            ["date"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ STRING }) }),
                  a_type({ typename = "function", args = TUPLE({ OS_DATE_TABLE_FORMAT, NUMBER }), rets = TUPLE({ OS_DATE_TABLE }) }),
                  a_type({ typename = "function", args = TUPLE({ OPT_STRING, OPT_NUMBER }), rets = TUPLE({ STRING }) }),
               },
            }),
            ["difftime"] = a_type({ typename = "function", args = TUPLE({ NUMBER, NUMBER }), rets = TUPLE({ NUMBER }) }),
            ["execute"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ BOOLEAN, STRING, INTEGER }) }),
            ["exit"] = a_type({ typename = "function", args = TUPLE({ UNION({ NUMBER, BOOLEAN }), BOOLEAN }), rets = TUPLE({}) }),
            ["getenv"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ STRING }) }),
            ["remove"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ BOOLEAN, STRING }) }),
            ["rename"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING }), rets = TUPLE({ BOOLEAN, STRING }) }),
            ["setlocale"] = a_type({ typename = "function", args = TUPLE({ STRING, OPT_STRING }), rets = TUPLE({ STRING }) }),
            ["time"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ OS_DATE_TABLE }), rets = TUPLE({ INTEGER }) }),
               },
            }),
            ["tmpname"] = a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ STRING }) }),
         },
      }),
      ["package"] = a_type({
         typename = "record",
         fields = {
            ["config"] = STRING,
            ["cpath"] = STRING,
            ["loaded"] = a_type({
               typename = "map",
               keys = STRING,
               values = ANY,
            }),
            ["loaders"] = a_type({
               typename = "array",
               elements = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ ANY }) }),
            }),
            ["loadlib"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING }), rets = TUPLE({ FUNCTION }) }),
            ["path"] = STRING,
            ["preload"] = TABLE,
            ["searchers"] = a_type({
               typename = "array",
               elements = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ ANY }) }),
            }),
            ["searchpath"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING, OPT_STRING, OPT_STRING }), rets = TUPLE({ STRING, STRING }) }),
         },
      }),
      ["string"] = a_type({
         typename = "record",
         fields = {
            ["byte"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ STRING, OPT_NUMBER }), rets = TUPLE({ INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ STRING, NUMBER, NUMBER }), rets = VARARG({ INTEGER }) }),
               },
            }),
            ["char"] = a_type({ typename = "function", args = VARARG({ NUMBER }), rets = TUPLE({ STRING }) }),
            ["dump"] = a_type({ typename = "function", args = TUPLE({ FUNCTION, OPT_BOOLEAN }), rets = TUPLE({ STRING }) }),
            ["find"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING, OPT_NUMBER, OPT_BOOLEAN }), rets = VARARG({ INTEGER, INTEGER, STRING }) }),
            ["format"] = a_type({ typename = "function", args = VARARG({ STRING, ANY }), rets = TUPLE({ STRING }) }),
            ["gmatch"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING }), rets = TUPLE({
               a_type({ typename = "function", args = TUPLE({}), rets = VARARG({ STRING }) }),
            }), }),
            ["gsub"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", args = TUPLE({ STRING, STRING, STRING, NUMBER }), rets = TUPLE({ STRING, INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ STRING, STRING, a_type({ typename = "map", keys = STRING, values = STRING }), NUMBER }), rets = TUPLE({ STRING, INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ STRING, STRING, a_type({ typename = "function", args = VARARG({ STRING }), rets = TUPLE({ STRING }) }) }), rets = TUPLE({ STRING, INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ STRING, STRING, a_type({ typename = "function", args = VARARG({ STRING }), rets = TUPLE({ NUMBER }) }) }), rets = TUPLE({ STRING, INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ STRING, STRING, a_type({ typename = "function", args = VARARG({ STRING }), rets = TUPLE({ BOOLEAN }) }) }), rets = TUPLE({ STRING, INTEGER }) }),
                  a_type({ typename = "function", args = TUPLE({ STRING, STRING, a_type({ typename = "function", args = VARARG({ STRING }), rets = TUPLE({}) }) }), rets = TUPLE({ STRING, INTEGER }) }),

               },
            }),
            ["len"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ INTEGER }) }),
            ["lower"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ STRING }) }),
            ["match"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING, NUMBER }), rets = VARARG({ STRING }) }),
            ["pack"] = a_type({ typename = "function", args = VARARG({ STRING, ANY }), rets = TUPLE({ STRING }) }),
            ["packsize"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ INTEGER }) }),
            ["rep"] = a_type({ typename = "function", args = TUPLE({ STRING, NUMBER }), rets = TUPLE({ STRING }) }),
            ["reverse"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ STRING }) }),
            ["sub"] = a_type({ typename = "function", args = TUPLE({ STRING, NUMBER, NUMBER }), rets = TUPLE({ STRING }) }),
            ["unpack"] = a_type({ typename = "function", args = TUPLE({ STRING, STRING, OPT_NUMBER }), rets = VARARG({ ANY }) }),
            ["upper"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({ STRING }) }),
         },
      }),
      ["table"] = a_type({
         typename = "record",
         fields = {
            ["concat"] = a_type({ typename = "function", args = TUPLE({ ARRAY_OF_STRING, OPT_STRING, OPT_NUMBER, OPT_NUMBER }), rets = TUPLE({ STRING }) }),
            ["insert"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ARRAY_OF_ALPHA, NUMBER, ALPHA }), rets = TUPLE({}) }),
                  a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ARRAY_OF_ALPHA, ALPHA }), rets = TUPLE({}) }),
               },
            }),
            ["move"] = a_type({
               typename = "poly",
               types = {
                  a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ARRAY_OF_ALPHA, NUMBER, NUMBER, NUMBER }), rets = TUPLE({ ARRAY_OF_ALPHA }) }),
                  a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ARRAY_OF_ALPHA, NUMBER, NUMBER, NUMBER, ARRAY_OF_ALPHA }), rets = TUPLE({ ARRAY_OF_ALPHA }) }),
               },
            }),
            ["pack"] = a_type({ typename = "function", args = VARARG({ ANY }), rets = TUPLE({ TABLE }) }),
            ["remove"] = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ARRAY_OF_ALPHA, OPT_NUMBER }), rets = TUPLE({ ALPHA }) }),
            ["sort"] = a_type({ typename = "function", typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ARRAY_OF_ALPHA, OPT_TABLE_SORT_FUNCTION }), rets = TUPLE({}) }),
            ["unpack"] = a_type({ typename = "function", needs_compat = true, typeargs = TUPLE({ ARG_ALPHA }), args = TUPLE({ ARRAY_OF_ALPHA, NUMBER, NUMBER }), rets = VARARG({ ALPHA }) }),
         },
      }),
      ["utf8"] = a_type({
         typename = "record",
         fields = {
            ["char"] = a_type({ typename = "function", args = VARARG({ NUMBER }), rets = TUPLE({ STRING }) }),
            ["charpattern"] = STRING,
            ["codepoint"] = a_type({ typename = "function", args = TUPLE({ STRING, OPT_NUMBER, OPT_NUMBER }), rets = VARARG({ INTEGER }) }),
            ["codes"] = a_type({ typename = "function", args = TUPLE({ STRING }), rets = TUPLE({
               a_type({ typename = "function", args = TUPLE({}), rets = TUPLE({ NUMBER, STRING }) }),
            }), }),
            ["len"] = a_type({ typename = "function", args = TUPLE({ STRING, NUMBER, NUMBER }), rets = TUPLE({ INTEGER }) }),
            ["offset"] = a_type({ typename = "function", args = TUPLE({ STRING, NUMBER, NUMBER }), rets = TUPLE({ INTEGER }) }),
         },
      }),
      ["_VERSION"] = STRING,
   }

   for _, t in pairs(standard_library) do
      fill_field_order(t)
      if is_typetype(t) then
         fill_field_order(t.def)
      end
   end
   fill_field_order(OS_DATE_TABLE)
   fill_field_order(DEBUG_GETINFO_TABLE)

   NOMINAL_FILE.found = standard_library["FILE"]
   NOMINAL_METATABLE_OF_ALPHA.found = standard_library["metatable"]

   for name, typ in pairs(standard_library) do
      globals[name] = { t = typ, needs_compat = stdlib_compat[name], is_const = true }
   end




   globals["@is_va"] = { t = ANY }

   if not is_first_init then
      last_typeid = save_typeid
   end

   return globals, standard_library
end

tl.init_env = function(lax, gen_compat, gen_target, predefined)
   if gen_compat == true or gen_compat == nil then
      gen_compat = "optional"
   elseif gen_compat == false then
      gen_compat = "off"
   end
   gen_compat = gen_compat

   if not gen_target then
      if _VERSION == "Lua 5.1" or _VERSION == "Lua 5.2" then
         gen_target = "5.1"
      else
         gen_target = "5.3"
      end
   end

   local globals, standard_library = init_globals(lax)

   local env = {
      ok = true,
      modules = {},
      loaded = {},
      loaded_order = {},
      globals = globals,
      gen_compat = gen_compat,
      gen_target = gen_target,
   }


   for name, var in pairs(standard_library) do
      if var.typename == "record" then
         env.modules[name] = var
      end
   end

   if predefined then
      for _, name in ipairs(predefined) do
         local module_type = require_module(name, lax, env)

         if module_type == INVALID then
            return nil, string.format("Error: could not predefine module '%s'", name)
         end
      end
   end

   return env
end

tl.type_check = function(ast, opts)
   opts = opts or {}
   local env = opts.env or tl.init_env(opts.lax, opts.gen_compat, opts.gen_target)
   local lax = opts.lax
   local filename = opts.filename

   local st = { env.globals }

   local symbol_list = {}
   local symbol_list_n = 0

   local all_needs_compat = {}

   local dependencies = {}
   local warnings = {}
   local errors = {}

   local module_type

   local function find_var(name, raw)
      for i = #st, 1, -1 do
         local scope = st[i]
         if scope[name] then
            if i == 1 and scope[name].needs_compat then
               all_needs_compat[name] = true
            end
            if not raw then
               scope[name].used = true
            end
            return scope[name]
         end
      end
   end

   local function simulate_g()

      local globals = {}
      for k, v in pairs(st[1]) do
         if k:sub(1, 1) ~= "@" then
            globals[k] = v.t
         end
      end
      return a_type({
         typename = "record",
         field_order = sorted_keys(globals),
         fields = globals,
      }), false
   end

   local function find_var_type(name, raw)
      local var = find_var(name, raw)
      if var then
         return var.t, var.is_const
      end
   end

   local function error_in_type(where, msg, ...)
      local n = select("#", ...)
      if n > 0 then
         local showt = {}
         for i = 1, n do
            local t = select(i, ...)
            if t.typename == "invalid" then
               return nil
            end
            showt[i] = show_type(t)
         end
         msg = msg:format(_tl_table_unpack(showt))
      end

      return {
         y = where.y,
         x = where.x,
         msg = msg,
         filename = where.filename or filename,
      }
   end

   local function type_error(t, msg, ...)
      local e = error_in_type(t, msg, ...)
      if e then
         table.insert(errors, e)
         return true
      else
         return false
      end
   end

   local function find_type(names, accept_typearg)
      local typ = find_var_type(names[1])
      if not typ then
         return nil
      end
      if typ.found then
         typ = typ.found
      end
      for i = 2, #names do
         local fields = typ.fields or (typ.def and typ.def.fields)
         if fields then
            typ = fields[names[i]]
            if typ == nil then
               return nil
            end
            if typ.found then
               typ = typ.found
            end
         else
            return nil
         end
      end
      if is_typetype(typ) or (accept_typearg and typ.typename == "typearg") then
         return typ
      end
   end

   local function union_type(t)
      if is_typetype(t) then
         return union_type(t.def)
      elseif t.typename == "tuple" then
         return union_type(t[1])
      elseif t.typename == "nominal" then
         local typetype = t.found or find_type(t.names)
         if not typetype then
            return "table"
         end
         return union_type(typetype)
      elseif t.typename == "record" then
         if t.is_userdata then
            return "userdata"
         end
         return "table"
      elseif table_types[t.typename] then
         return "table"
      else
         return t.typename
      end
   end

   local function is_valid_union(typ)


      local n_table_types = 0
      local n_function_types = 0
      local n_userdata_types = 0
      local n_string_enum = 0
      local has_primitive_string_type = false
      for _, t in ipairs(typ.types) do
         local ut = union_type(t)
         if ut == "userdata" then
            n_userdata_types = n_userdata_types + 1
            if n_userdata_types > 1 then
               return false, "cannot discriminate a union between multiple userdata types: %s"
            end
         elseif ut == "table" then
            n_table_types = n_table_types + 1
            if n_table_types > 1 then
               return false, "cannot discriminate a union between multiple table types: %s"
            end
         elseif ut == "function" then
            n_function_types = n_function_types + 1
            if n_function_types > 1 then
               return false, "cannot discriminate a union between multiple function types: %s"
            end
         elseif ut == "enum" or (ut == "string" and not has_primitive_string_type) then
            n_string_enum = n_string_enum + 1
            if n_string_enum > 1 then
               return false, "cannot discriminate a union between multiple string/enum types: %s"
            end
            if ut == "string" then
               has_primitive_string_type = true
            end
         end
      end
      return true
   end

   local function resolve_typetype(t)
      if is_typetype(t) then
         return t.def
      else
         return t
      end
   end

   local no_nested_types = {
      ["string"] = true,
      ["number"] = true,
      ["integer"] = true,
      ["boolean"] = true,
      ["thread"] = true,
      ["any"] = true,
      ["enum"] = true,
      ["nil"] = true,
      ["unknown"] = true,
   }

   local function resolve_typevars(typ)
      local errs
      local seen = {}

      local function resolve(t)


         if no_nested_types[t.typename] or (t.typename == "nominal" and not t.typevals) then
            return t
         end

         seen = seen or {}
         if seen[t] then
            return seen[t]
         end

         local orig_t = t
         if t.typename == "typevar" then
            t = find_var_type(t.typevar)
            local rt
            if not t then
               rt = orig_t
            elseif t.typename == "string" then

               rt = STRING
            elseif no_nested_types[t.typename] or
               (t.typename == "nominal" and not t.typevals) then
               rt = t
            end
            if rt then
               seen[orig_t] = rt
               return rt
            end
         end

         local copy = {}
         seen[orig_t] = copy

         copy.typename = t.typename
         copy.filename = t.filename
         copy.typeid = t.typeid
         copy.x = t.x
         copy.y = t.y
         copy.yend = t.yend
         copy.xend = t.xend
         copy.names = t.names

         for i, tf in ipairs(t) do
            copy[i] = resolve(tf)
         end

         if t.typename == "array" then
            copy.elements = resolve(t.elements)

         elseif t.typename == "typearg" then
            copy.typearg = t.typearg
         elseif t.typename == "typevar" then
            copy.typevar = t.typevar
         elseif is_typetype(t) then
            copy.def = resolve(t.def)
         elseif t.typename == "nominal" then
            copy.typevals = resolve(t.typevals)
            copy.found = t.found
         elseif t.typename == "function" then
            if t.typeargs then
               copy.typeargs = {}
               for i, tf in ipairs(t.typeargs) do
                  copy.typeargs[i] = resolve(tf)
               end
            end

            copy.is_method = t.is_method
            copy.args = resolve(t.args)
            copy.rets = resolve(t.rets)
         elseif t.typename == "record" or t.typename == "arrayrecord" then
            if t.typeargs then
               copy.typeargs = {}
               for i, tf in ipairs(t.typeargs) do
                  copy.typeargs[i] = resolve(tf)
               end
            end

            if t.elements then
               copy.elements = resolve(t.elements)
            end

            copy.fields = {}
            for _, k in ipairs(t.field_order) do
               copy.fields[k] = resolve(t.fields[k])
            end
            copy.field_order = t.field_order

            if t.meta_fields then
               copy.meta_fields = {}
               for _, k in ipairs(t.meta_field_order) do
                  copy.meta_fields[k] = resolve(t.meta_fields[k])
               end
               copy.meta_field_order = t.meta_field_order
            end
         elseif t.typename == "map" then
            copy.keys = resolve(t.keys)
            copy.values = resolve(t.values)
         elseif t.typename == "union" then
            copy.types = {}
            for i, tf in ipairs(t.types) do
               copy.types[i] = resolve(tf)
            end

            local ok, err = is_valid_union(copy)
            if not ok then
               errs = errs or {}
               table.insert(errs, error_in_type(t, err, t))
            end
         elseif t.typename == "poly" or t.typename == "tupletable" then
            copy.types = {}
            for i, tf in ipairs(t.types) do
               copy.types[i] = resolve(tf)
            end
         elseif t.typename == "tuple" then
            copy.is_va = t.is_va
         end

         return copy
      end

      local copy = resolve(typ)
      if errs then
         return false, INVALID, errs
      end
      return true, copy
   end

   local function infer_var(emptytable, t, node)
      local is_global = (emptytable.declared_at and emptytable.declared_at.kind == "global_declaration")
      local nst = is_global and 1 or #st
      for i = nst, 1, -1 do
         local scope = st[i]
         if scope[emptytable.assigned_to] then
            scope[emptytable.assigned_to] = {
               t = t,
               is_const = false,
            }
            t.inferred_at = node
            t.inferred_at_file = filename
         end
      end
   end

   local function find_global(name)
      local scope = st[1]
      if scope[name] then
         return scope[name].t, scope[name].is_const
      end
   end

   local function resolve_tuple(t)
      if t.typename == "tuple" then
         t = t[1]
      end
      if t == nil then
         return NIL
      end
      return t
   end

   local function node_warning(tag, node, fmt, ...)
      table.insert(warnings, {
         y = node.y,
         x = node.x,
         msg = fmt:format(...),
         filename = filename,
         tag = tag,
      })
   end

   local function node_error(node, msg, ...)
      type_error(node, msg, ...)
      node.type = INVALID
      return node.type
   end

   local function terr(t, s, ...)
      return { error_in_type(t, s, ...) }
   end

   local function add_unknown(node, name)
      node_warning("unknown", node, "unknown variable: %s", name)
   end

   local function redeclaration_warning(node, old_var)
      if node.tk:sub(1, 1) == "_" then return end
      if old_var.declared_at then
         node_warning("redeclaration", node, "redeclaration of variable '%s' (originally declared at %d:%d)", node.tk, old_var.declared_at.y, old_var.declared_at.x)
      else
         node_warning("redeclaration", node, "redeclaration of variable '%s'", node.tk)
      end
   end

   local function check_if_redeclaration(new_name, at)
      local old = find_var(new_name, true)
      if old then
         redeclaration_warning(at, old)
      end
   end

   local function unused_warning(name, var)
      local prefix = name:sub(1, 1)
      if var.declared_at and
         not var.is_narrowed and
         prefix ~= "_" and
         prefix ~= "@" then

         if name:sub(1, 2) == "::" then
            node_warning("unused", var.declared_at, "unused label %s", name)
         else
            node_warning(
            "unused",
            var.declared_at,
            "unused %s %s: %s",
            var.is_func_arg and "argument" or
            var.t.typename == "function" and "function" or
            is_typetype(var.t) and "type" or
            "variable",
            name,
            show_type(var.t))

         end
      end
   end

   local function shallow_copy(t)
      local copy = {}
      for k, v in pairs(t) do
         copy[k] = v
      end
      return copy
   end

   local function reserve_symbol_list_slot(node)
      symbol_list_n = symbol_list_n + 1
      node.symbol_list_slot = symbol_list_n
   end

   local function add_var(node, var, valtype, is_const, is_narrowing, dont_check_redeclaration)
      if lax and node and is_unknown(valtype) and (var ~= "self" and var ~= "...") and not is_narrowing then
         add_unknown(node, var)
      end
      local scope = st[#st]
      local old_var = scope[var]
      if not is_const then
         valtype = shallow_copy(valtype)
         valtype.tk = nil
      end
      if old_var and is_narrowing then
         if not old_var.is_narrowed then
            old_var.narrowed_from = old_var.t
         end
         old_var.is_narrowed = true
         old_var.t = valtype
      else
         if not dont_check_redeclaration and
            node and
            not is_narrowing and
            var ~= "self" and
            var ~= "..." and
            var:sub(1, 1) ~= "@" then

            check_if_redeclaration(var, node)
         end
         scope[var] = { t = valtype, is_const = is_const, is_narrowed = is_narrowing, declared_at = node }
         if old_var then


            if not old_var.used then
               unused_warning(var, old_var)
            end
         end
      end

      if node and valtype.typename ~= "unresolved" and valtype.typename ~= "none" then
         node.type = node.type or valtype
         local slot
         if node.symbol_list_slot then
            slot = node.symbol_list_slot
         else
            symbol_list_n = symbol_list_n + 1
            slot = symbol_list_n
         end
         symbol_list[slot] = { y = node.y, x = node.x, name = var, typ = assert(scope[var].t) }
      end

      return scope[var]
   end

   local CompareTypes = {}

   local function compare_and_infer_typevars(t1, t2, comp)

      if t1.typevar == t2.typevar then
         return true
      end


      local typevar = t2.typevar or t1.typevar


      local vt = find_var_type(typevar)
      if vt then

         if t2.typevar then
            return comp(t1, vt)
         else
            return comp(vt, t2)
         end
      else

         local other = t2.typevar and t1 or t2
         local ok, resolved, errs = resolve_typevars(other)
         if not ok then
            return false, errs
         end
         if resolved.typename ~= "unknown" then
            resolved = resolve_typetype(resolved)
            add_var(nil, typevar, resolved)
         end
         return true
      end
   end

   local function add_errs_prefixing(src, dst, prefix, node)
      if not src then
         return
      end
      for _, err in ipairs(src) do
         err.msg = prefix .. err.msg


         if node and node.y and (
            (err.filename ~= filename) or
            (not err.y) or
            (node.y > err.y or (node.y == err.y and node.x > err.x))) then

            err.y = node.y
            err.x = node.x
            err.filename = filename
         end

         table.insert(dst, err)
      end
   end

   local is_a

   local TypeGetter = {}

   local function match_record_fields(t1, t2, cmp)
      cmp = cmp or is_a
      local fielderrs = {}
      for _, k in ipairs(t1.field_order) do
         local f = t1.fields[k]
         local t2k = t2(k)
         if t2k == nil then
            if not lax then
               table.insert(fielderrs, error_in_type(f, "unknown field " .. k))
            end
         else
            local __, errs = cmp(f, t2k)
            add_errs_prefixing(errs, fielderrs, "record field doesn't match: " .. k .. ": ")
         end
      end
      if #fielderrs > 0 then
         return false, fielderrs
      end
      return true
   end

   local function match_fields_to_record(t1, t2, cmp)
      local ok, fielderrs = match_record_fields(t1, function(k) return t2.fields[k] end, cmp)
      if not ok then
         local errs = {}
         add_errs_prefixing(errs, fielderrs, show_type(t1) .. " is not a " .. show_type(t2) .. ": ")
         return false, errs
      end
      return true
   end

   local function match_fields_to_map(t1, t2)
      if not match_record_fields(t1, function(_) return t2.values end) then
         return false, { error_in_type(t1, "record is not a valid map; not all fields have the same type") }
      end
      return true
   end

   local function arg_check(cmp, a, b, at, n, errs)
      local matches, match_errs = cmp(a, b)
      if not matches then
         add_errs_prefixing(match_errs, errs, "argument " .. n .. ": ", at)
         return false
      end
      return true
   end

   local same_type

   local function has_all_types_of(t1s, t2s, cmp)
      for _, t1 in ipairs(t1s) do
         local found = false
         for _, t2 in ipairs(t2s) do
            if cmp(t2, t1) then
               found = true
               break
            end
         end
         if not found then
            return false
         end
      end
      return true
   end

   local function any_errors(all_errs)
      if #all_errs == 0 then
         return true
      else
         return false, all_errs
      end
   end

   local function close_nested_records(t)
      for _, ft in pairs(t.fields) do
         if is_typetype(ft) then
            ft.closed = true
            if is_record_type(ft.def) then
               close_nested_records(ft.def)
            end
         end
      end
   end

   local function close_types(vars)
      for _, var in pairs(vars) do
         if is_typetype(var.t) then
            var.t.closed = true
            if is_record_type(var.t.def) then
               close_nested_records(var.t.def)
            end
         end
      end
   end

   local Unused = {}






   local function check_for_unused_vars(vars)
      local list = {}
      for name, var in pairs(vars) do
         if var.declared_at and not var.used then
            table.insert(list, { y = var.declared_at.y, x = var.declared_at.x, name = name, var = var })
         end
      end
      if list[1] then
         table.sort(list, function(a, b)
            return a.y < a.y or (a.y == b.y and a.x < b.x)
         end)
         for _, u in ipairs(list) do
            unused_warning(u.name, u.var)
         end
      end
   end

   local function begin_scope(node)
      table.insert(st, {})

      if node then
         symbol_list_n = symbol_list_n + 1
         symbol_list[symbol_list_n] = { y = node.y, x = node.x, name = "@{" }
      end
   end

   local function end_scope(node)
      local unresolved = st[#st]["@unresolved"]
      if unresolved then
         local upper = st[#st - 1]["@unresolved"]
         if upper then
            for name, nodes in pairs(unresolved.t.labels) do
               for _, n in ipairs(nodes) do
                  upper.t.labels[name] = upper.t.labels[name] or {}
                  table.insert(upper.t.labels[name], n)
               end
            end
            for name, types in pairs(unresolved.t.nominals) do
               for _, typ in ipairs(types) do
                  upper.t.nominals[name] = upper.t.nominals[name] or {}
                  table.insert(upper.t.nominals[name], typ)
               end
            end
         else
            st[#st - 1]["@unresolved"] = unresolved
         end
      end
      close_types(st[#st])
      check_for_unused_vars(st[#st])
      table.remove(st)

      if node then
         if symbol_list[symbol_list_n].name == "@{" then
            symbol_list[symbol_list_n] = nil
            symbol_list_n = symbol_list_n - 1
         else
            symbol_list_n = symbol_list_n + 1
            symbol_list[symbol_list_n] = { y = assert(node.yend), x = assert(node.xend), name = "@}" }
         end
      end
   end

   local end_scope_and_none_type = function(node, _children)
      end_scope(node)
      node.type = NONE
      return node.type
   end

   local function resolve_typevars_at(t, where)
      assert(where)
      local ok, typ, errs = resolve_typevars(t)
      if not ok then
         assert(where.y)
         add_errs_prefixing(errs, errors, "", where)
      end
      return typ
   end

   local resolve_nominal
   do
      local function match_typevals(t, def)
         if t.typevals and def.typeargs then
            if #t.typevals ~= #def.typeargs then
               type_error(t, "mismatch in number of type arguments")
               return nil
            end

            begin_scope()
            for i, tt in ipairs(t.typevals) do
               add_var(nil, def.typeargs[i].typearg, tt)
            end
            local ret = resolve_typevars_at(def, t)
            end_scope()
            return ret
         elseif t.typevals then
            type_error(t, "spurious type arguments")
            return nil
         elseif def.typeargs then
            type_error(t, "missing type arguments in %s", def)
            return nil
         else
            return def
         end
      end

      resolve_nominal = function(t)
         if t.resolved then
            return t.resolved
         end

         local resolved

         local typetype = t.found or find_type(t.names)
         if not typetype then
            type_error(t, "unknown type %s", t)
         elseif is_typetype(typetype) then
            if typetype.is_alias then
               typetype = typetype.def.found
               assert(is_typetype(typetype))
            end
            assert(typetype.def.typename ~= "nominal")
            resolved = match_typevals(t, typetype.def)
         else
            type_error(t, table.concat(t.names, ".") .. " is not a type")
         end

         if not resolved then
            resolved = a_type({ typename = "bad_nominal", names = t.names })
         end

         if not t.filename then
            t.filename = resolved.filename
            if t.x == nil and t.y == nil then
               t.x = resolved.x
               t.y = resolved.y
            end
         end
         t.found = typetype
         t.resolved = resolved
         return resolved
      end
   end

   local function are_same_nominals(t1, t2)
      local same_names
      if t1.found and t2.found then
         same_names = t1.found.typeid == t2.found.typeid
      else
         local ft1 = t1.found or find_type(t1.names)
         local ft2 = t2.found or find_type(t2.names)
         if ft1 and ft2 then
            same_names = ft1.typeid == ft2.typeid
         else
            if not ft1 then
               type_error(t1, "unknown type %s", t1)
            end
            if not ft2 then
               type_error(t2, "unknown type %s", t2)
            end
            return false, {}
         end
      end

      if same_names then
         if t1.typevals == nil and t2.typevals == nil then
            return true
         elseif t1.typevals and t2.typevals and #t1.typevals == #t2.typevals then
            local all_errs = {}
            for i = 1, #t1.typevals do
               local _, errs = same_type(t1.typevals[i], t2.typevals[i])
               add_errs_prefixing(errs, all_errs, "type parameter <" .. show_type(t2.typevals[i]) .. ">: ", t1)
            end
            if #all_errs == 0 then
               return true
            else
               return false, all_errs
            end
         end
      else
         local t1name = show_type(t1)
         local t2name = show_type(t2)
         if t1name == t2name then
            local t1r = resolve_nominal(t1)
            if t1r.filename then
               t1name = t1name .. " (defined in " .. t1r.filename .. ":" .. t1r.y .. ")"
            end
            local t2r = resolve_nominal(t2)
            if t2r.filename then
               t2name = t2name .. " (defined in " .. t2r.filename .. ":" .. t2r.y .. ")"
            end
         end
         return false, terr(t1, t1name .. " is not a " .. t2name)
      end
   end

   local is_known_table_type
   local resolve_tuple_and_nominal = nil


   same_type = function(t1, t2)
      assert(type(t1) == "table")
      assert(type(t2) == "table")

      if t1.typename == "typevar" or t2.typename == "typevar" then
         return compare_and_infer_typevars(t1, t2, same_type)
      end

      if t1.typename == "emptytable" and is_known_table_type(resolve_tuple_and_nominal(t2)) then
         return true
      end

      if t1.typename ~= t2.typename then
         return false, terr(t1, "got %s, expected %s", t1, t2)
      end
      if t1.typename == "array" then
         return same_type(t1.elements, t2.elements)
      elseif t1.typename == "tupletable" then
         local all_errs = {}
         for i = 1, math.min(#t1.types, #t2.types) do
            local ok, err = same_type(t1.types[i], t2.types[i])
            if not ok then
               add_errs_prefixing(err, all_errs, "values", t1)
            end
         end
         return any_errors(all_errs)
      elseif t1.typename == "map" then
         local all_errs = {}
         local k_ok, k_errs = same_type(t1.keys, t2.keys)
         if not k_ok then
            add_errs_prefixing(k_errs, all_errs, "keys", t1)
         end
         local v_ok, v_errs = same_type(t1.values, t2.values)
         if not v_ok then
            add_errs_prefixing(v_errs, all_errs, "values", t1)
         end
         return any_errors(all_errs)
      elseif t1.typename == "union" then
         if has_all_types_of(t1.types, t2.types, same_type) and
            has_all_types_of(t2.types, t1.types, same_type) then
            return true
         else
            return false, terr(t1, "got %s, expected %s", t1, t2)
         end
      elseif t1.typename == "nominal" then
         return are_same_nominals(t1, t2)
      elseif t1.typename == "record" then
         return match_fields_to_record(t1, t2, same_type) and
         match_fields_to_record(t2, t1, same_type)
      elseif t1.typename == "function" then
         if #t1.args ~= #t2.args then
            return false, terr(t1, "different number of input arguments: got " .. #t1.args .. ", expected " .. #t2.args)
         end
         if #t1.rets ~= #t2.rets then
            return false, terr(t1, "different number of return values: got " .. #t1.args .. ", expected " .. #t2.args)
         end
         local all_errs = {}
         for i = 1, #t1.args do
            arg_check(same_type, t1.args[i], t2.args[i], t1, i, all_errs)
         end
         for i = 1, #t1.rets do
            local _, errs = same_type(t1.rets[i], t2.rets[i])
            add_errs_prefixing(errs, all_errs, "return " .. i, t1)
         end
         return any_errors(all_errs)
      elseif t1.typename == "arrayrecord" then
         local ok, errs = same_type(t1.elements, t2.elements)
         if not ok then
            return ok, errs
         end
         return match_fields_to_record(t1, t2, same_type) and
         match_fields_to_record(t2, t1, same_type)
      end
      return true
   end

   local function unite(types, flatten_constants)
      if #types == 1 then
         return types[1]
      end

      local ts = {}
      local stack = {}


      local types_seen = {}

      types_seen[NIL.typeid] = true
      types_seen["nil"] = true

      local i = 1
      while types[i] or stack[1] do
         local t
         if stack[1] then
            t = table.remove(stack)
         else
            t = types[i]
            i = i + 1
         end
         t = resolve_tuple(t)
         if t.typename == "union" then
            for _, s in ipairs(t.types) do
               table.insert(stack, s)
            end
         else
            if primitive[t.typename] and (flatten_constants or not t.tk) then
               if not types_seen[t.typename] then
                  types_seen[t.typename] = true
                  table.insert(ts, t)
               end
            else
               local typeid = t.typeid
               if t.typename == "nominal" then
                  typeid = resolve_nominal(t).typeid
               end
               if not types_seen[typeid] then
                  types_seen[typeid] = true
                  table.insert(ts, t)
               end
            end
         end
      end

      if #ts == 1 then
         return ts[1]
      else
         return a_type({
            typename = "union",
            types = ts,
         })
      end
   end

   local function combine_errs(...)
      local errs
      for i = 1, select("#", ...) do
         local e = select(i, ...)
         if e then
            errs = errs or {}
            for _, err in ipairs(e) do
               table.insert(errs, err)
            end
         end
      end
      if not errs then
         return true
      else
         return false, errs
      end
   end

   local known_table_types = {
      array = true,
      map = true,
      record = true,
      arrayrecord = true,
      tupletable = true,
   }
   is_known_table_type = function(t)
      return known_table_types[t.typename]
   end

   local expand_type
   local function arraytype_from_tuple(where, tupletype)

      local element_type = unite(tupletype.types)
      local valid = element_type.typename ~= "union" and true or is_valid_union(element_type)
      if valid then
         return a_type({
            elements = element_type,
            typename = "array",
         })
      end


      local arr_type = a_type({
         elements = tupletype.types[1],
         typename = "array",
      })
      for i = 2, #tupletype.types do
         arr_type = expand_type(where, arr_type, a_type({ elements = tupletype.types[i], typename = "array" }))
         if not arr_type or not arr_type.elements then
            return nil, terr(tupletype, "unable to convert tuple %s to array", tupletype)
         end
      end
      return arr_type
   end


   is_a = function(t1, t2, for_equality)
      assert(type(t1) == "table")
      assert(type(t2) == "table")

      if lax and (is_unknown(t1) or is_unknown(t2)) then
         return true
      end

      if t1.typename == "bad_nominal" or t2.typename == "bad_nominal" then
         return false
      end


      if t1.typename == "nil" then
         return true
      end

      if t2.typename ~= "tuple" then
         t1 = resolve_tuple(t1)
      end

      if t2.typename == "tuple" and t1.typename ~= "tuple" then
         t1 = a_type({
            typename = "tuple",
            [1] = t1,
         })
      end

      if t1.typename == "typevar" or t2.typename == "typevar" then
         return compare_and_infer_typevars(t1, t2, is_a)
      end


      if t2.typename == "any" then
         return true




      elseif t1.typename == "union" then
         for _, t in ipairs(t1.types) do
            if not is_a(t, t2, for_equality) then
               return false, terr(t1, "got %s, expected %s", t1, t2)
            end
         end
         return true




      elseif t2.typename == "union" then
         for _, t in ipairs(t2.types) do
            if is_a(t1, t, for_equality) then
               return true
            end
         end




      elseif t2.typename == "poly" then
         for _, t in ipairs(t2.types) do
            if not is_a(t1, t, for_equality) then
               return false, terr(t1, "cannot match against all alternatives of the polymorphic type")
            end
         end
         return true




      elseif t1.typename == "poly" then
         for _, t in ipairs(t1.types) do
            if is_a(t, t2, for_equality) then
               return true
            end
         end
         return false, terr(t1, "cannot match against any alternatives of the polymorphic type")
      elseif t1.typename == "nominal" and t2.typename == "nominal" then
         local same, err = are_same_nominals(t1, t2)
         if same then
            return true
         end
         local t1r = resolve_tuple_and_nominal(t1)
         local t2r = resolve_tuple_and_nominal(t2)
         if is_record_type(t1r) and is_record_type(t2r) then
            return same, err
         else
            return is_a(t1r, t2r, for_equality)
         end
      elseif t1.typename == "enum" and t2.typename == "string" then
         local ok
         if for_equality then
            ok = t2.tk and t1.enumset[unquote(t2.tk)]
         else
            ok = true
         end
         if ok then
            return true
         else
            return false, terr(t1, "enum is incompatible with %s", t2)
         end
      elseif t1.typename == "integer" and t2.typename == "number" then
         return true
      elseif t1.typename == "string" and t2.typename == "enum" then
         local ok = t1.tk and t2.enumset[unquote(t1.tk)]
         if ok then
            return true
         else
            if t1.tk then
               return false, terr(t1, "%s is not a member of %s", t1, t2)
            else
               return false, terr(t1, "string is not a %s", t2)
            end
         end
      elseif t1.typename == "nominal" or t2.typename == "nominal" then
         local t1r = resolve_tuple_and_nominal(t1)
         local t2r = resolve_tuple_and_nominal(t2)
         local ok, errs = is_a(t1r, t2r, for_equality)
         if errs and #errs == 1 then
            if errs[1].msg:match("^got ") then


               errs = terr(t1, "got %s, expected %s", t1, t2)
            end
         end
         return ok, errs
      elseif t1.typename == "emptytable" and is_known_table_type(t2) then
         return true
      elseif t2.typename == "array" then
         if is_array_type(t1) then
            if is_a(t1.elements, t2.elements) then
               return true
            end
         elseif t1.typename == "tupletable" then
            if t2.inferred_len and t2.inferred_len > #t1.types then
               return false, terr(t1, "incompatible length, expected maximum length of " .. tostring(#t1.types) .. ", got " .. tostring(t2.inferred_len))
            end
            local t1a, err = arraytype_from_tuple(t1.inferred_at, t1)
            if not t1a then
               return false, err
            end
            if not is_a(t1a, t2) then
               return false, terr(t2, "got %s (from %s), expected %s", t1a, t1, t2)
            end
            return true
         elseif t1.typename == "map" then
            local _, errs_keys, errs_values
            _, errs_keys = is_a(t1.keys, INTEGER)
            _, errs_values = is_a(t1.values, t2.elements)
            return combine_errs(errs_keys, errs_values)
         end
      elseif t2.typename == "record" then
         if is_record_type(t1) then
            return match_fields_to_record(t1, t2)
         elseif is_typetype(t1) and is_record_type(t1.def) then
            return is_a(t1.def, t2, for_equality)
         end
      elseif t2.typename == "arrayrecord" then
         if t1.typename == "array" then
            return is_a(t1.elements, t2.elements)
         elseif t1.typename == "tupletable" then
            if t2.inferred_len and t2.inferred_len > #t1.types then
               return false, terr(t1, "incompatible length, expected maximum length of " .. tostring(#t1.types) .. ", got " .. tostring(t2.inferred_len))
            end
            local t1a, err = arraytype_from_tuple(t1.inferred_at, t1)
            if not t1a then
               return false, err
            end
            if not is_a(t1a, t2) then
               return false, terr(t2, "got %s (from %s), expected %s", t1a, t1, t2)
            end
            return true
         elseif t1.typename == "record" then
            return match_fields_to_record(t1, t2)
         elseif t1.typename == "arrayrecord" then
            if not is_a(t1.elements, t2.elements) then
               return false, terr(t1, "array parts have incompatible element types")
            end
            return match_fields_to_record(t1, t2)
         elseif is_typetype(t1) and is_record_type(t1.def) then
            return is_a(t1.def, t2, for_equality)
         end
      elseif t2.typename == "map" then
         if t1.typename == "map" then
            local _, errs_keys, errs_values
            if t2.keys.typename ~= "any" then
               _, errs_keys = same_type(t2.keys, t1.keys)
            end
            if t2.values.typename ~= "any" then
               _, errs_values = same_type(t1.values, t2.values)
            end
            return combine_errs(errs_keys, errs_values)
         elseif t1.typename == "array" or t1.typename == "tupletable" then
            local elements
            if t1.typename == "tupletable" then
               local arr_type = arraytype_from_tuple(t1.inferred_at, t1)
               if not arr_type then
                  return false, terr(t1, "Unable to convert tuple %s to map", t1)
               end
               elements = arr_type.elements
            else
               elements = t1.elements
            end
            local _, errs_keys, errs_values
            _, errs_keys = is_a(INTEGER, t2.keys)
            _, errs_values = is_a(elements, t2.values)
            return combine_errs(errs_keys, errs_values)
         elseif is_record_type(t1) then
            if not is_a(t2.keys, STRING) then
               return false, terr(t1, "can't match a record to a map with non-string keys")
            end
            if t2.keys.typename == "enum" then
               for _, k in ipairs(t1.field_order) do
                  if not t2.keys.enumset[k] then
                     return false, terr(t1, "key is not an enum value: " .. k)
                  end
               end
            end
            return match_fields_to_map(t1, t2)
         end
      elseif t2.typename == "tupletable" then
         if t1.typename == "tupletable" then
            for i = 1, math.min(#t1.types, #t2.types) do
               if not is_a(t1.types[i], t2.types[i], for_equality) then
                  return false, terr(t1, "in tuple entry " .. tostring(i) .. ": got %s, expected %s", t1.types[i], t2.types[i])
               end
            end
            if for_equality and #t1.types ~= #t2.types then
               return false, terr(t1, "tuples are not the same size")
            end
            if #t1.types > #t2.types then
               return false, terr(t1, "tuple %s is too big for tuple %s", t1, t2)
            end
            return true
         elseif is_array_type(t1) then
            if t1.inferred_len and t1.inferred_len > #t2.types then
               return false, terr(t1, "incompatible length, expected maximum length of " .. tostring(#t2.types) .. ", got " .. tostring(t1.inferred_len))
            end



            local len = (t1.inferred_len and t1.inferred_len > 0) and
            t1.inferred_len or
            #t2.types

            for i = 1, len do
               if not is_a(t1.elements, t2.types[i], for_equality) then
                  return false, terr(t1, "tuple entry " .. tostring(i) .. " of type %s does not match type of array elements, which is %s", t2.types[i], t1.elements)
               end
            end
            return true
         end
      elseif t1.typename == "function" and t2.typename == "function" then
         local all_errs = {}
         if (not t2.args.is_va) and #t1.args > #t2.args then
            table.insert(all_errs, error_in_type(t1, "incompatible number of arguments: got " .. #t1.args .. " %s, expected " .. #t2.args .. " %s", t1.args, t2.args))
         else
            for i = (t1.is_method and 2 or 1), #t1.args do
               arg_check(is_a, t1.args[i], t2.args[i] or ANY, nil, i, all_errs)
            end
         end
         local diff_by_va = #t2.rets - #t1.rets == 1 and t2.rets.is_va
         if #t1.rets < #t2.rets and not diff_by_va then
            table.insert(all_errs, error_in_type(t1, "incompatible number of returns: got " .. #t1.rets .. " %s, expected " .. #t2.rets .. " %s", t1.rets, t2.rets))
         else
            local nrets = #t2.rets
            if diff_by_va then
               nrets = nrets - 1
            end
            for i = 1, nrets do
               local _, errs = is_a(t1.rets[i], t2.rets[i])
               add_errs_prefixing(errs, all_errs, "return " .. i .. ": ")
            end
         end
         if #all_errs == 0 then
            return true
         else
            return false, all_errs
         end
      elseif lax and ((not for_equality) and t2.typename == "boolean") then

         return true
      elseif t1.typename == t2.typename then
         return true
      end

      return false, terr(t1, "got %s, expected %s", t1, t2)
   end

   local function assert_is_a(node, t1, t2, context, name)
      t1 = resolve_tuple(t1)
      t2 = resolve_tuple(t2)
      if lax and (is_unknown(t1) or is_unknown(t2)) then
         return true
      end


      if t1.typename == "nil" then
         return true
      elseif t2.typename == "unresolved_emptytable_value" then
         if is_number_type(t2.emptytable_type.keys) then
            infer_var(t2.emptytable_type, a_type({ typename = "array", elements = t1 }), node)
         else
            infer_var(t2.emptytable_type, a_type({ typename = "map", keys = t2.emptytable_type.keys, values = t1 }), node)
         end
         return true
      elseif t2.typename == "emptytable" then
         if is_known_table_type(t1) then
            infer_var(t2, shallow_copy(t1), node)
         elseif t1.typename ~= "emptytable" then
            node_error(node, context .. ": " .. (name and (name .. ": ") or "") .. "assigning %s to a variable declared with {}", t1)
         end
         return true
      end

      local ok, match_errs = is_a(t1, t2)
      add_errs_prefixing(match_errs, errors, context .. ": " .. (name and (name .. ": ") or ""), node)
      return ok
   end

   local unknown_dots = {}

   local function add_unknown_dot(node, name)
      if not unknown_dots[name] then
         unknown_dots[name] = true
         add_unknown(node, name)
      end
   end

   local type_check_function_call
   do
      local function resolve_for_call(node, func, args, is_method)

         if lax and is_unknown(func) then
            func = a_type({ typename = "function", args = VARARG({ UNKNOWN }), rets = VARARG({ UNKNOWN }) })
            if node.e1.op and node.e1.op.op == ":" and node.e1.e1.kind == "variable" then
               add_unknown_dot(node, node.e1.e1.tk .. "." .. node.e1.e2.tk)
            end
         end

         func = resolve_tuple_and_nominal(func)
         if func.typename ~= "function" and func.typename ~= "poly" then

            if is_typetype(func) and func.def.typename == "record" then
               func = func.def
            end

            if func.meta_fields and func.meta_fields["__call"] then
               table.insert(args, 1, func)
               func = func.meta_fields["__call"]
               is_method = true
            end
         end
         return func, is_method
      end

      local function mark_invalid_typeargs(f)
         if f.typeargs then
            for _, a in ipairs(f.typeargs) do
               if not find_var(a.typearg) then
                  add_var(nil, a.typearg, lax and UNKNOWN or INVALID)
               end
            end
         end
      end

      local function try_match_func_args(node, f, args, argdelta)
         local errs = {}

         local given = #args
         local expected = #f.args
         local va = f.args.is_va
         local nargs = va and
         math.max(given, expected) or
         math.min(given, expected)

         for a = 1, nargs do
            local argument = args[a]
            local farg = f.args[a] or (va and f.args[expected])
            if argument == nil then
               if va then
                  break
               end
            else
               local at = node.e2 and node.e2[a] or node
               if not arg_check(is_a, argument, farg, at, (a + argdelta), errs) then
                  return nil, errs
               end
            end
         end

         mark_invalid_typeargs(f)


         for a = 1, given do
            local argument = args[a]
            if argument.typename == "emptytable" then
               local farg = f.args[a] or (va and f.args[expected])
               local where = node.e2[a + argdelta] or node.e2
               infer_var(argument, resolve_typevars_at(farg, where), where)
            end
         end

         return resolve_typevars_at(f.rets, node)
      end

      local function revert_typeargs(func)
         if func.typeargs then
            for _, fnarg in ipairs(func.typeargs) do
               if st[#st][fnarg.typearg] then
                  st[#st][fnarg.typearg] = nil
               end
            end
         end
      end

      local function fail_call(node, func, nargs, errs)
         if errs then

            for _, err in ipairs(errs) do
               table.insert(errors, err)
            end
         else

            local expects = {}
            if func.typename == "poly" then
               for _, f in ipairs(func.types) do
                  table.insert(expects, tostring(#f.args or 0))
               end
               table.sort(expects)
               for i = #expects, 1, -1 do
                  if expects[i] == expects[i + 1] then
                     table.remove(expects, i)
                  end
               end
            else
               table.insert(expects, tostring(#func.args or 0))
            end
            node_error(node, "wrong number of arguments (given " .. nargs .. ", expects " .. table.concat(expects, " or ") .. ")")
         end

         local f = func.typename == "poly" and func.types[1] or func
         mark_invalid_typeargs(f)
         return resolve_typevars_at(f.rets, node)
      end

      local function check_call(node, func, args, is_method, argdelta)
         assert(type(func) == "table")
         assert(type(args) == "table")

         if not (func.typename == "function" or func.typename == "poly") then
            func, is_method = resolve_for_call(node, func, args, is_method)
         end

         argdelta = is_method and -1 or argdelta or 0

         local is_func = func.typename == "function"
         local is_poly = func.typename == "poly"
         if not (is_func or is_poly) then
            return node_error(node, "not a function: %s", func)
         end

         local passes, n = 1, 1
         if is_poly then
            passes, n = 3, #func.types
         end

         local given = #args
         local tried
         local first_errs
         for pass = 1, passes do
            for i = 1, n do
               if (not tried) or not tried[i] then
                  local f = is_func and func or func.types[i]
                  if f.is_method and not is_method and not (args[1] and is_a(args[1], f.args[1])) then
                     return node_error(node, "invoked method as a regular function: use ':' instead of '.'")
                  end
                  local expected = #f.args


                  if (is_func and (given <= expected or (f.args.is_va and given > expected))) or

                     (is_poly and ((pass == 1 and given == expected) or

                     (pass == 2 and given < expected) or

                     (pass == 3 and f.args.is_va and given > expected))) then

                     local matched, errs = try_match_func_args(node, f, args, argdelta)
                     if matched then

                        return matched
                     end
                     first_errs = first_errs or errs

                     if is_poly then
                        tried = tried or {}
                        tried[i] = true
                        revert_typeargs(f)
                     end
                  end
               end
            end
         end

         return fail_call(node, func, given, first_errs)
      end

      type_check_function_call = function(node, func, args, is_method, argdelta)
         begin_scope()
         local ret = check_call(node, func, args, is_method, argdelta)
         end_scope()
         return ret
      end
   end

   local function match_record_key(node, tbl, key, orig_tbl)
      assert(type(tbl) == "table")
      assert(type(key) == "table")

      tbl = resolve_tuple_and_nominal(tbl)
      local type_description = tbl.typename
      if tbl.typename == "string" or tbl.typename == "enum" then
         tbl = find_var_type("string")
      end

      if lax and (is_unknown(tbl) or tbl.typename == "typevar") then
         if node.e1.kind == "variable" and node.op.op ~= "@funcall" then
            add_unknown_dot(node, node.e1.tk .. "." .. key.tk)
         end
         return UNKNOWN
      end

      if tbl.is_alias then
         return node_error(key, "cannot use a nested type alias as a concrete value")
      end

      tbl = resolve_typetype(tbl)

      if tbl.typename == "emptytable" then
      elseif is_record_type(tbl) then
         assert(tbl.fields, "record has no fields!?")

         if key.kind == "string" or key.kind == "identifier" then
            if tbl.fields[key.tk] then
               return tbl.fields[key.tk]
            end
         end
      else
         if is_unknown(tbl) then
            if not lax then
               node_error(key, "cannot index a value of unknown type")
            end
         else
            node_error(key, "cannot index something that is not a record: %s", tbl)
         end
         return INVALID
      end

      if lax then
         if node.e1.kind == "variable" and node.op.op ~= "@funcall" then
            add_unknown_dot(node, node.e1.tk .. "." .. key.tk)
         end
         return UNKNOWN
      end

      local description
      if node.e1.kind == "variable" then
         description = type_description .. " '" .. node.e1.tk .. "' of type " .. show_type(resolve_tuple(orig_tbl))
      else
         description = "type " .. show_type(resolve_tuple(orig_tbl))
      end

      return node_error(key, "invalid key '" .. key.tk .. "' in " .. description)
   end

   local function widen_in_scope(scope, var)
      if scope[var].is_narrowed then
         if scope[var].narrowed_from then
            scope[var].t = scope[var].narrowed_from
            scope[var].narrowed_from = nil
            scope[var].is_narrowed = false
         else
            scope[var] = nil
         end
         return true
      end
      return false
   end

   local function widen_back_var(var)
      local widened = false
      for i = #st, 1, -1 do
         if st[i][var] then
            if widen_in_scope(st[i], var) then
               widened = true
            else
               break
            end
         end
      end
      return widened
   end

   local function widen_all_unions()
      for i = #st, 1, -1 do
         for var, _ in pairs(st[i]) do
            widen_in_scope(st[i], var)
         end
      end
   end

   local function add_global(node, var, valtype, is_const)
      if lax and is_unknown(valtype) and (var ~= "self" and var ~= "...") then
         add_unknown(node, var)
      end
      st[1][var] = { t = valtype, is_const = is_const }
      if node then
         node.type = node.type or valtype
      end
   end

   local function get_rets(rets)
      if lax and (#rets == 0) then
         return VARARG({ UNKNOWN })
      end
      local t = rets
      if not t.typename then
         t = TUPLE(t)
      end
      assert(t.typeid)
      return t
   end

   local function add_internal_function_variables(node)
      add_var(nil, "@is_va", node.args.type.is_va and ANY or NIL)

      add_var(nil, "@return", node.rets or a_type({ typename = "tuple" }))
   end

   local function add_function_definition_for_recursion(node)
      local args = a_type({ typename = "tuple" })
      for _, fnarg in ipairs(node.args) do
         table.insert(args, fnarg.type)
      end

      add_var(nil, node.name.tk, a_type({
         typename = "function",
         args = args,
         rets = get_rets(node.rets),
      }))
   end

   local function fail_unresolved()
      local unresolved = st[#st]["@unresolved"]
      if unresolved then
         st[#st]["@unresolved"] = nil
         for name, nodes in pairs(unresolved.t.labels) do
            for _, node in ipairs(nodes) do
               node_error(node, "no visible label '" .. name .. "' for goto")
            end
         end
         for _, types in pairs(unresolved.t.nominals) do
            for _, typ in ipairs(types) do
               assert(typ.x)
               assert(typ.y)
               type_error(typ, "unknown type %s", typ)
            end
         end
      end
   end

   local function end_function_scope(node)
      fail_unresolved()
      end_scope(node)
   end

   resolve_tuple_and_nominal = function(t)
      t = resolve_tuple(t)
      if t.typename == "nominal" then
         t = resolve_nominal(t)
      end
      assert(t.typename ~= "nominal")
      return t
   end

   local function flatten_list(list)
      local exps = {}
      for i = 1, #list - 1 do
         table.insert(exps, resolve_tuple_and_nominal(list[i]))
      end
      if #list > 0 then
         local last = list[#list]
         if last.typename == "tuple" then
            for _, val in ipairs(last) do
               table.insert(exps, val)
            end
         else
            table.insert(exps, last)
         end
      end
      return exps
   end

   local function get_assignment_values(vals, wanted)
      local ret = {}
      if vals == nil then
         return ret
      end


      local is_va = vals.is_va
      for i = 1, #vals - 1 do
         ret[i] = resolve_tuple(vals[i])
      end

      local last = vals[#vals]
      if last.typename == "tuple" then

         is_va = last.is_va
         for _, v in ipairs(last) do
            table.insert(ret, v)
         end
      else

         table.insert(ret, last)
      end


      if is_va and last and #ret < wanted then
         while #ret < wanted do
            table.insert(ret, last)
         end
      end
      return ret
   end

   local function match_all_record_field_names(node, a, field_names, errmsg)
      local t
      for _, k in ipairs(field_names) do
         local f = a.fields[k]
         if not t then
            t = f
         else
            if not same_type(f, t) then
               t = nil
               break
            end
         end
      end
      if t then
         return t
      else
         return node_error(node, errmsg)
      end
   end

   local function type_check_index(node, idxnode, a, b)
      local orig_a = a
      local orig_b = b
      a = resolve_tuple_and_nominal(a)
      b = resolve_tuple_and_nominal(b)

      if a.typename == "tupletable" and is_a(b, INTEGER) then
         if idxnode.constnum then
            if idxnode.constnum > #a.types or
               idxnode.constnum < 1 or
               idxnode.constnum ~= math.floor(idxnode.constnum) then

               return node_error(idxnode, "index " .. tostring(idxnode.constnum) .. " out of range for tuple %s", a)
            end
            return a.types[idxnode.constnum]
         else
            local array_type = arraytype_from_tuple(idxnode, a)
            if not array_type then
               type_error(a, "cannot index this tuple with a variable because it would produce a union type that cannot be discriminated at runtime")
               return INVALID
            end
            return array_type.elements
         end
      elseif is_array_type(a) and is_a(b, INTEGER) then
         return a.elements
      elseif a.typename == "emptytable" then
         if a.keys == nil then
            a.keys = resolve_tuple(orig_b)
            a.keys_inferred_at = assert(node)
            a.keys_inferred_at_file = filename
         else
            if not is_a(b, a.keys) then
               local inferred = " (type of keys inferred at " .. a.keys_inferred_at_file .. ":" .. a.keys_inferred_at.y .. ":" .. a.keys_inferred_at.x .. ": )"
               return node_error(idxnode, "inconsistent index type: %s, expected %s" .. inferred, orig_b, a.keys)
            end
         end
         return a_type({ y = node.y, x = node.x, typename = "unresolved_emptytable_value", emptytable_type = a })
      elseif a.typename == "map" then
         if is_a(b, a.keys) then
            return a.values
         else
            return node_error(idxnode, "wrong index type: %s, expected %s", orig_b, a.keys)
         end
      elseif node.e2.kind == "string" or node.e2.kind == "enum_item" then
         return match_record_key(node, a, { y = node.e2.y, x = node.e2.x, kind = "string", tk = assert(node.e2.conststr) }, orig_a)
      elseif is_record_type(a) then
         if b.typename == "enum" then
            local field_names = sorted_keys(b.enumset)
            for _, k in ipairs(field_names) do
               if not a.fields[k] then
                  return node_error(idxnode, "enum value '" .. k .. "' is not a field in %s", a)
               end
            end
            return match_all_record_field_names(idxnode, a, field_names,
            "cannot index, not all enum values map to record fields of the same type")
         elseif is_a(b, STRING) then
            return node_error(idxnode, "cannot index object of type %s with a string, consider using an enum", orig_a)
         end
      end
      if lax and is_unknown(a) then
         return UNKNOWN
      else
         return node_error(idxnode, "cannot index object of type %s with %s", orig_a, orig_b)
      end
   end

   expand_type = function(where, old, new)
      if not old or old.typename == "nil" then
         return new
      else
         if not is_a(new, old) then
            if old.typename == "map" and is_record_type(new) then
               if old.keys.typename == "string" then
                  for _, ftype in fields_of(new) do
                     old.values = expand_type(where, old.values, ftype)
                  end
               else
                  node_error(where, "cannot determine table literal type")
               end
            elseif is_record_type(old) and is_record_type(new) then
               old.typename = "map"
               old.keys = STRING
               for _, ftype in fields_of(old) do
                  if not old.values then
                     old.values = ftype
                  else
                     old.values = expand_type(where, old.values, ftype)
                  end
               end
               for _, ftype in fields_of(new) do
                  if not old.values then
                     new.values = ftype
                  else
                     new.values = expand_type(where, old.values, ftype)
                  end
               end
               old.fields = nil
               old.field_order = nil
            elseif old.typename == "union" then
               new.tk = nil
               table.insert(old.types, new)
            else
               old.tk = nil
               new.tk = nil
               return unite({ old, new })
            end
         end
      end
      return old
   end

   local function find_record_to_extend(exp)
      if exp.kind == "type_identifier" then
         local t = find_var_type(exp.tk)

         if t.def then
            if not t.def.closed and not t.closed then
               return t.def
            end
         end
         if not t.closed then
            return t
         end
      elseif exp.kind == "op" and exp.op.op == "." then
         local t = find_record_to_extend(exp.e1)
         if not t then
            return nil
         end
         while exp.e2.kind == "op" and exp.e2.op.op == "." do
            t = t.fields and t.fields[exp.e2.e1.tk]
            if not t then
               return nil
            end
            exp = exp.e2
         end
         t = t.fields and t.fields[exp.e2.tk]
         return t
      end
   end


   local facts_and
   local facts_or
   local facts_not
   local apply_facts
   local FACT_TRUTHY
   do
      setmetatable(Fact, {
         __call = function(_, fact)
            return setmetatable(fact, {
               __tostring = function(f)
                  if f.fact == "is" then
                     return ("(%s is %s)"):format(f.var, show_type(f.typ))
                  elseif f.fact == "==" then
                     return ("(%s == %s)"):format(f.var, show_type(f.typ))
                  elseif f.fact == "truthy" then
                     return "*"
                  elseif f.fact == "not" then
                     return ("(not %s)"):format(tostring(f.f1))
                  elseif f.fact == "or" then
                     return ("(%s or %s)"):format(tostring(f.f1), tostring(f.f2))
                  elseif f.fact == "and" then
                     return ("(%s and %s)"):format(tostring(f.f1), tostring(f.f2))
                  end
               end,
            })
         end,
      })

      FACT_TRUTHY = Fact({ fact = "truthy" })

      facts_and = function(f1, f2, where)
         return Fact({ fact = "and", f1 = f1, f2 = f2, where = where })
      end

      facts_or = function(f1, f2, where)
         if f1 and f2 then
            return Fact({ fact = "or", f1 = f1, f2 = f2, where = where })
         else
            return nil
         end
      end

      facts_not = function(f1, where)
         if f1 then
            return Fact({ fact = "not", f1 = f1, where = where })
         else
            return nil
         end
      end


      local function unite_types(t1, t2)
         return unite({ t2, t1 })
      end


      local function intersect_types(t1, t2)
         if t2.typename == "union" then
            t1, t2 = t2, t1
         end
         if t1.typename == "union" then
            local out = {}
            for _, t in ipairs(t1.types) do
               if is_a(t, t2) then
                  table.insert(out, t)
               end
            end
            return unite(out)
         else
            if is_a(t1, t2) then
               return t1
            elseif is_a(t2, t1) then
               return t2
            else
               return INVALID
            end
         end
      end

      local function resolve_if_union(t)
         local rt = resolve_tuple_and_nominal(t)
         if rt.typename == "union" then
            return rt
         end
         return t
      end


      local function subtract_types(t1, t2)
         local types = {}

         t1 = resolve_if_union(t1)


         if t1.typename ~= "union" then
            return t1
         end

         t2 = resolve_if_union(t2)
         local t2types = t2.types or { t2 }

         for _, at in ipairs(t1.types) do
            local not_present = true
            for _, bt in ipairs(t2types) do
               if same_type(at, bt) then
                  not_present = false
                  break
               end
            end
            if not_present then
               table.insert(types, at)
            end
         end

         if #types == 0 then
            return INVALID
         end

         return unite(types)
      end

      local eval_not
      local not_facts
      local or_facts
      local and_facts
      local eval_fact

      local function invalid_from(f)
         return Fact({ fact = "is", var = f.var, typ = INVALID, where = f.where })
      end

      not_facts = function(fs)
         local ret = {}
         for var, f in pairs(fs) do
            local typ = find_var_type(f.var, true)
            local fact = "=="
            local where = f.where
            if not typ then
               typ = INVALID
            else
               if f.fact == "is" then
                  if typ.typename == "typevar" then

                     where = nil
                  elseif not is_a(f.typ, typ) then
                     node_warning("branch", f.where, f.var .. " (of type %s) can never be a %s", show_type(typ), show_type(f.typ))
                     typ = INVALID
                  else
                     fact = "is"
                     typ = subtract_types(typ, f.typ)
                  end
               elseif f.fact == "==" then

                  where = nil
               end
            end
            ret[var] = Fact({ fact = fact, var = var, typ = typ, where = where })
         end
         return ret
      end

      eval_not = function(f)
         if not f then
            return {}
         elseif f.fact == "is" then
            return not_facts({ [f.var] = f })
         elseif f.fact == "not" then
            return eval_fact(f.f1)
         elseif f.fact == "and" and f.f2 and f.f2.fact == "truthy" then
            return eval_not(f.f1)
         elseif f.fact == "or" and f.f2 and f.f2.fact == "truthy" then
            return eval_fact(f.f1)
         elseif f.fact == "and" then
            return or_facts(not_facts(eval_fact(f.f1)), not_facts(eval_fact(f.f2)))
         elseif f.fact == "or" then
            return and_facts(not_facts(eval_fact(f.f1)), not_facts(eval_fact(f.f2)))
         else
            return not_facts(eval_fact(f))
         end
      end

      or_facts = function(fs1, fs2)
         local ret = {}

         for var, f in pairs(fs2) do
            if fs1[var] then
               local fact = (fs1[var].fact == "is" and f.fact == "is") and
               "is" or "=="
               ret[var] = Fact({ fact = fact, var = var, typ = unite_types(f.typ, fs1[var].typ), where = f.where })
            end
         end

         return ret
      end

      and_facts = function(fs1, fs2)
         local ret = {}
         local has = {}

         for var, f in pairs(fs1) do
            local rt
            local fact
            if fs2[var] then
               fact = (fs2[var].fact == "is" and f.fact == "is") and "is" or "=="
               rt = intersect_types(f.typ, fs2[var].typ)
            else
               fact = "=="
               rt = f.typ
            end
            ret[var] = Fact({ fact = fact, var = var, typ = rt, where = f.where })
            has[fact] = true
         end

         for var, f in pairs(fs2) do
            if not fs1[var] then
               ret[var] = Fact({ fact = "==", var = var, typ = f.typ, where = f.where })
               has["=="] = true
            end
         end

         if has["is"] and has["=="] then
            for _, f in pairs(ret) do
               f.fact = "=="
            end
         end

         return ret
      end

      eval_fact = function(f)
         if not f then
            return {}
         elseif f.fact == "is" then
            local typ = find_var_type(f.var, true)
            if not typ then
               return { [f.var] = invalid_from(f) }
            end
            if typ.typename ~= "typevar" and is_a(typ, f.typ) then
               node_warning("branch", f.where, f.var .. " (of type %s) is always a %s", show_type(typ), show_type(f.typ))
               return { [f.var] = f }
            elseif typ.typename ~= "typevar" and not is_a(f.typ, typ) then
               node_error(f.where, f.var .. " (of type %s) can never be a %s", typ, f.typ)
               return { [f.var] = invalid_from(f) }
            else
               return { [f.var] = f }
            end
         elseif f.fact == "==" then
            return { [f.var] = f }
         elseif f.fact == "not" then
            return eval_not(f.f1)
         elseif f.fact == "truthy" then
            return {}
         elseif f.fact == "and" and f.f2 and f.f2.fact == "truthy" then
            return eval_fact(f.f1)
         elseif f.fact == "or" and f.f2 and f.f2.fact == "truthy" then
            return eval_not(f.f1)
         elseif f.fact == "and" then
            return and_facts(eval_fact(f.f1), eval_fact(f.f2))
         elseif f.fact == "or" then
            return or_facts(eval_fact(f.f1), eval_fact(f.f2))
         end
      end

      apply_facts = function(where, known)
         if not known then
            return
         end

         local facts = eval_fact(known)

         for v, f in pairs(facts) do
            if f.typ.typename == "invalid" then
               node_error(where, "cannot resolve a type for " .. v .. " here")
            end
            local t = shallow_copy(f.typ)
            t.inferred_at = f.where and where
            t.inferred_at_file = filename
            add_var(nil, v, t, true, true)
         end
      end
   end

   local function dismiss_unresolved(name)
      local unresolved = st[#st]["@unresolved"]
      if unresolved then
         if unresolved.t.nominals[name] then
            for _, t in ipairs(unresolved.t.nominals[name]) do
               resolve_nominal(t)
            end
         end
         unresolved.t.nominals[name] = nil
      end
   end

   local type_check_funcall

   local function special_pcall_xpcall(node, _a, b, argdelta)
      local base_nargs = (node.e1.tk == "xpcall") and 2 or 1
      if #node.e2 < base_nargs then
         node_error(node, "wrong number of arguments (given " .. #node.e2 .. ", expects at least " .. base_nargs .. ")")
         return TUPLE({ BOOLEAN })
      end

      local ftype = table.remove(b, 1)
      local fe2 = {}
      if node.e1.tk == "xpcall" then
         base_nargs = 2
         local msgh = table.remove(b, 1)
         assert_is_a(node.e2[2], msgh, XPCALL_MSGH_FUNCTION, "in message handler")
      end
      for i = base_nargs + 1, #node.e2 do
         table.insert(fe2, node.e2[i])
      end
      local fnode = {
         y = node.y,
         x = node.x,
         kind = "op",
         op = { op = "@funcall" },
         e1 = node.e2[1],
         e2 = fe2,
      }
      local rets = type_check_funcall(fnode, ftype, b, argdelta + base_nargs)
      if rets.typename ~= "tuple" then
         rets = a_type({ typename = "tuple", rets })
      end
      table.insert(rets, 1, BOOLEAN)
      return rets
   end

   local special_functions = {
      ["rawget"] = function(node, _a, b, _argdelta)

         if #b == 2 then
            local b1 = resolve_tuple_and_nominal(b[1])
            local b2 = resolve_tuple_and_nominal(b[2])
            local knode = node.e2[2]
            if is_record_type(b1) and knode.conststr then
               return match_record_key(node, b1, { y = knode.y, x = knode.x, kind = "string", tk = assert(knode.conststr) }, b1)
            else
               return type_check_index(node, knode, b1, b2)
            end
         else
            node_error(node, "rawget expects two arguments")
            return INVALID
         end
      end,

      ["print_type"] = function(node, _a, b, _argdelta)

         if #b == 0 then

            print("-----------------------------------------")
            for i, scope in ipairs(st) do
               for s, v in pairs(scope) do
                  print(("%2d %-14s %-11s %s"):format(i, s, v.t.typename, show_type(v.t):sub(1, 50)))
               end
            end
            print("-----------------------------------------")
            return NONE
         else
            local t = show_type(b[1])
            print(t)
            node_warning("debug", node.e2[1], "type is: %s", t)
            return b
         end
      end,

      ["require"] = function(node, _a, b, _argdelta)
         if #b ~= 1 then
            return node_error(node, "require expects one literal argument")
         end
         if node.e2[1].kind ~= "string" then
            return node_error(node, "don't know how to resolve a dynamic require")
         end

         local module_name = assert(node.e2[1].conststr)
         local t, found = require_module(module_name, lax, env)
         if not found then
            return node_error(node, "module not found: '" .. module_name .. "'")
         end

         if t.typename == "invalid" then
            if lax then
               return UNKNOWN
            end
            return node_error(node, "no type information for required module: '" .. module_name .. "'")
         end

         dependencies[module_name] = t.filename
         return t
      end,

      ["pcall"] = special_pcall_xpcall,
      ["xpcall"] = special_pcall_xpcall,

      ["assert"] = function(node, a, b, argdelta)
         node.known = FACT_TRUTHY
         return type_check_function_call(node, a, b, false, argdelta)
      end,
   }

   type_check_funcall = function(node, a, b, argdelta)
      argdelta = argdelta or 0
      if node.e1.kind == "variable" then
         local special = special_functions[node.e1.tk]
         if special then
            return special(node, a, b, argdelta)
         else
            return type_check_function_call(node, a, b, false, argdelta)
         end
      elseif node.e1.op and node.e1.op.op == ":" then
         table.insert(b, 1, node.e1.e1.type)
         return type_check_function_call(node, a, b, true)
      else
         return type_check_function_call(node, a, b, false, argdelta)
      end
   end


   local function is_localizing_a_variable(node, i)
      return node.exps and
      node.exps[i] and
      node.exps[i].kind == "variable" and
      node.exps[i].tk == node.vars[i].tk
   end

   local function resolve_nominal_typetype(typetype)
      if typetype.def.typename == "nominal" then
         if typetype.def.typevals then
            typetype.def = resolve_nominal(typetype.def)
            typetype.def.typeargs = nil
         else
            local names = typetype.def.names
            local found = find_type(names)
            if (not found) or (not is_typetype(found)) then
               type_error(typetype, "%s is not a type", typetype)
               found = a_type({ typename = "bad_nominal", names = names })
            end
            return found, true
         end
      end
      return typetype, false
   end

   local function missing_initializer(node, i, name)
      if lax then
         return UNKNOWN
      else
         if node.exps then
            node_error(node.vars[i], "assignment in declaration did not produce an initial value for variable '" .. name .. "'")
         else
            node_error(node.vars[i], "variable '" .. name .. "' has no type or initial value")
         end
         return INVALID
      end
   end

   local function set_expected_types_to_decltypes(node, children)
      local decls = node.kind == "assignment" and children[1] or node.decltype
      if decls and node.exps then
         local ndecl = #decls
         local nexps = #node.exps
         for i = 1, nexps do
            local typ
            typ = decls[i]
            if typ then
               if i == nexps and ndecl > nexps then
                  typ = a_type({ y = node.y, x = node.x, filename = filename, typename = "tuple", types = {} })
                  for a = i, ndecl do
                     table.insert(typ.types, decls[a])
                  end
               end
               node.exps[i].expected = typ
               node.exps[i].expected_context = { kind = node.kind, name = node.vars[i].tk }
            end
         end
      end
   end

   local function is_positive_int(n)
      return n and n >= 1 and math.floor(n) == n
   end

   local context_name = {
      ["local_declaration"] = "in local declaration",
      ["global_declaration"] = "in global declaration",
      ["assignment"] = "in assignment",
   }

   local function in_context(ctx, msg)
      if not ctx then
         return msg
      end
      local where = context_name[ctx.kind]
      if where then
         return where .. ": " .. (ctx.name and ctx.name .. ": " or "") .. msg
      else
         return msg
      end
   end

   local function check_redeclared_key(ctx, where, seen_keys, ck, n)
      local key = ck or n
      if key then
         local s = seen_keys[key]
         if s then
            node_error(where, in_context(ctx, "redeclared key " .. tostring(key) .. " (previously declared at " .. filename .. ":" .. s.y .. ":" .. s.x .. ")"))
         else
            seen_keys[key] = where
         end
      end
   end

   local function infer_table_literal(node, children)
      local typ = a_type({
         filename = filename,
         y = node.y,
         x = node.x,
         typename = "emptytable",
      })

      local is_record = false
      local is_array = false
      local is_map = false

      local is_tuple = false
      local is_not_tuple = false

      local last_array_idx = 1
      local largest_array_idx = -1

      local seen_keys = {}

      for i, child in ipairs(children) do
         assert(child.typename == "table_item")

         local ck = child.kname
         local n = node[i].key.constnum
         check_redeclared_key(nil, node[i], seen_keys, ck, n)

         local uvtype = resolve_tuple(child.vtype)
         if ck then
            is_record = true
            if not typ.fields then
               typ.fields = {}
               typ.field_order = {}
            end
            typ.fields[ck] = uvtype
            table.insert(typ.field_order, ck)
         elseif is_number_type(child.ktype) then
            is_array = true
            if not is_not_tuple then
               is_tuple = true
            end
            if not typ.types then
               typ.types = {}
            end

            if node[i].key_parsed == "implicit" then
               if i == #children and child.vtype.typename == "tuple" then

                  for _, c in ipairs(child.vtype) do
                     typ.elements = expand_type(node, typ.elements, c)
                     typ.types[last_array_idx] = resolve_tuple(c)
                     last_array_idx = last_array_idx + 1
                  end
               else
                  typ.types[last_array_idx] = uvtype
                  last_array_idx = last_array_idx + 1
                  typ.elements = expand_type(node, typ.elements, uvtype)
               end
            else
               if not is_positive_int(n) then
                  typ.elements = expand_type(node, typ.elements, uvtype)
                  is_not_tuple = true
               elseif n then
                  typ.types[n] = uvtype
                  if n > largest_array_idx then
                     largest_array_idx = n
                  end
                  typ.elements = expand_type(node, typ.elements, uvtype)
               end
            end

            if last_array_idx > largest_array_idx then
               largest_array_idx = last_array_idx
            end
            if not typ.elements then
               is_array = false
            end
         else
            is_map = true
            child.ktype.tk = nil
            typ.keys = expand_type(node, typ.keys, child.ktype)
            typ.values = expand_type(node, typ.values, uvtype)
         end
      end

      if is_array and is_map then
         typ.typename = "map"
         typ.keys = expand_type(node, typ.keys, INTEGER)
         typ.values = expand_type(node, typ.values, typ.elements)
         typ.elements = nil
         node_error(node, "cannot determine type of table literal")
      elseif is_record and is_array then
         typ.typename = "arrayrecord"
      elseif is_record and is_map then
         if typ.keys.typename == "string" then
            typ.typename = "map"
            for _, ftype in fields_of(typ) do
               typ.values = expand_type(node, typ.values, ftype)
            end
            typ.fields = nil
            typ.field_order = nil
         else
            node_error(node, "cannot determine type of table literal")
         end
      elseif is_array then
         if is_not_tuple then
            typ.typename = "array"
            typ.inferred_len = largest_array_idx - 1
         else
            local pure_array = true

            local last_t
            for _, current_t in pairs(typ.types) do
               if last_t then
                  if not same_type(last_t, current_t) then
                     pure_array = false
                     break
                  end
               end
               last_t = current_t
            end

            if not pure_array then
               typ.typename = "tupletable"
            else
               typ.typename = "array"
               typ.inferred_len = largest_array_idx - 1
            end
         end
      elseif is_record then
         typ.typename = "record"
      elseif is_map then
         typ.typename = "map"
      elseif is_tuple then
         typ.typename = "tupletable"
         if not typ.types or #typ.types == 0 then
            node_error(node, "cannot determine type of tuple elements")
         end
      end

      return typ
   end

   local visit_node = {}

   visit_node.cbs = {
      ["statements"] = {
         before = function(node)
            begin_scope(node)
         end,
         after = function(node, _children)

            if #st == 2 then
               fail_unresolved()
            end

            if not node.is_repeat then
               end_scope(node)
            end

            node.type = NONE
            return node.type
         end,
      },
      ["local_type"] = {
         before = function(node)
            node.value.type, node.value.is_alias = resolve_nominal_typetype(node.value.newtype)
            add_var(node.var, node.var.tk, node.value.type, node.var.is_const)
         end,
         after = function(node, _children)
            dismiss_unresolved(node.var.tk)
            node.type = NONE
            return node.type
         end,
      },
      ["global_type"] = {
         before = function(node)
            node.value.newtype, node.value.is_alias = resolve_nominal_typetype(node.value.newtype)
            add_global(node.var, node.var.tk, node.value.newtype, node.var.is_const)
         end,
         after = function(node, _children)
            local existing, existing_is_const = find_global(node.var.tk)
            local var = node.var
            if existing then
               if existing_is_const == true and not var.is_const then
                  node_error(var, "global was previously declared as <const>: " .. var.tk)
               end
               if existing_is_const == false and var.is_const then
                  node_error(var, "global was previously declared as not <const>: " .. var.tk)
               end
               if not same_type(existing, node.value.newtype) then
                  node_error(var, "cannot redeclare global with a different type: previous type of " .. var.tk .. " is %s", existing)
               end
            end
            dismiss_unresolved(var.tk)
            node.type = NONE
            return node.type
         end,
      },
      ["local_declaration"] = {
         before = function(node)
            for _, var in ipairs(node.vars) do
               reserve_symbol_list_slot(var)
            end
         end,
         before_expressions = set_expected_types_to_decltypes,
         after = function(node, children)
            local vals = get_assignment_values(children[3], #node.vars)
            for i, var in ipairs(node.vars) do
               local decltype = node.decltype and node.decltype[i]
               local infertype = vals and vals[i]
               if lax and infertype and infertype.typename == "nil" then
                  infertype = nil
               end
               if decltype and infertype then
                  assert_is_a(node.vars[i], infertype, decltype, "in local declaration", var.tk)
               end
               local t = decltype or infertype
               if t == nil then
                  t = missing_initializer(node, i, var.tk)
               elseif t.typename == "emptytable" then
                  t.declared_at = node
                  t.assigned_to = var.tk
               end
               t.inferred_len = nil

               assert(var)
               add_var(var, var.tk, t, var.is_const, is_localizing_a_variable(node, i))

               dismiss_unresolved(var.tk)
            end
            node.type = NONE
            return node.type
         end,
      },
      ["global_declaration"] = {
         before_expressions = set_expected_types_to_decltypes,
         after = function(node, children)
            local vals = get_assignment_values(children[3], #node.vars)
            for i, var in ipairs(node.vars) do
               local decltype = node.decltype and node.decltype[i]
               local infertype = vals and vals[i]
               if lax and infertype and infertype.typename == "nil" then
                  infertype = nil
               end
               if decltype and infertype then
                  assert_is_a(node.vars[i], infertype, decltype, "in global declaration", var.tk)
               end
               local t = decltype or infertype
               local existing, existing_is_const = find_global(var.tk)
               if existing then
                  if infertype and existing_is_const then
                     node_error(var, "cannot reassign to <const> global: " .. var.tk)
                  end
                  if existing_is_const == true and not var.is_const then
                     node_error(var, "global was previously declared as <const>: " .. var.tk)
                  end
                  if existing_is_const == false and var.is_const then
                     node_error(var, "global was previously declared as not <const>: " .. var.tk)
                  end
                  if t and not same_type(existing, t) then
                     node_error(var, "cannot redeclare global with a different type: previous type of " .. var.tk .. " is %s", existing)
                  end
               else
                  if t == nil then
                     t = missing_initializer(node, i, var.tk)
                  elseif t.typename == "emptytable" then
                     t.declared_at = node
                     t.assigned_to = var.tk
                  end
                  t.inferred_len = nil
                  add_global(var, var.tk, t, var.is_const)
                  var.type = t

                  dismiss_unresolved(var.tk)
               end
            end
            node.type = NONE
            return node.type
         end,
      },
      ["assignment"] = {
         before_expressions = set_expected_types_to_decltypes,
         after = function(node, children)
            local vals = get_assignment_values(children[3], #children[1])
            local exps = flatten_list(vals)
            for i, vartype in ipairs(children[1]) do
               local varnode = node.vars[i]
               local is_const = varnode.is_const
               if varnode.kind == "variable" then
                  if widen_back_var(varnode.tk) then
                     vartype, is_const = find_var_type(varnode.tk)
                  end
               end
               if is_const then
                  node_error(varnode, "cannot assign to <const> variable")
               end
               if vartype then
                  local val = exps[i]
                  if is_typetype(resolve_tuple_and_nominal(vartype)) then
                     node_error(varnode, "cannot reassign a type")
                  elseif val then
                     assert_is_a(varnode, val, vartype, "in assignment")
                     if varnode.kind == "variable" and vartype.typename == "union" then

                        add_var(varnode, varnode.tk, val, false, true)
                     end
                  else
                     node_error(varnode, "variable is not being assigned a value")
                     if #node.exps == 1 and node.exps[1].kind == "op" and node.exps[1].op.op == "@funcall" then
                        local rets = node.exps[1].type
                        if rets.typename == "tuple" then
                           local msg = #rets == 1 and
                           "only 1 value is returned by the function" or
                           ("only " .. #rets .. " values are returned by the function")
                           node_warning("hint", varnode, msg)
                        end
                     end
                  end
               else
                  node_error(varnode, "unknown variable")
               end
            end
            node.type = NONE
            return node.type
         end,
      },
      ["if"] = {
         after = function(node, _children)
            node.type = NONE
            return node.type
         end,
      },
      ["if_block"] = {
         before = function(node)
            begin_scope(node)
            if node.if_block_n > 1 then
               local ifnode = node.if_parent
               local f = facts_not(ifnode.if_blocks[1].exp.known, node)
               for e = 2, node.if_block_n - 1 do
                  f = facts_and(f, facts_not(ifnode.if_blocks[e].exp.known, node), node)
               end
               apply_facts(node, f)
            end
         end,
         before_statements = function(node)
            if node.exp then
               apply_facts(node.exp, node.exp.known)
            end
         end,
         after = end_scope_and_none_type,
      },
      ["while"] = {
         before = function()

            widen_all_unions()
         end,
         before_statements = function(node)
            begin_scope(node)
            apply_facts(node.exp, node.exp.known)
         end,
         after = end_scope_and_none_type,
      },
      ["label"] = {
         before = function(node)

            widen_all_unions()
            local label_id = "::" .. node.label .. "::"
            if st[#st][label_id] then
               node_error(node, "label '" .. node.label .. "' already defined at " .. filename)
            end
            local unresolved = st[#st]["@unresolved"]
            node.type = a_type({ y = node.y, x = node.x, typename = "none" })
            local var = add_var(node, label_id, node.type)
            if unresolved then
               if unresolved.t.labels[node.label] then
                  var.used = true
               end
               unresolved.t.labels[node.label] = nil
            end
         end,
      },
      ["goto"] = {
         after = function(node, _children)
            if not find_var_type("::" .. node.label .. "::") then
               local unresolved = st[#st]["@unresolved"] and st[#st]["@unresolved"].t
               if not unresolved then
                  unresolved = { typename = "unresolved", labels = {}, nominals = {} }
                  add_var(node, "@unresolved", unresolved)
               end
               unresolved.labels[node.label] = unresolved.labels[node.label] or {}
               table.insert(unresolved.labels[node.label], node)
            end
            node.type = NONE
            return node.type
         end,
      },
      ["repeat"] = {
         before = function()

            widen_all_unions()
         end,

         after = end_scope_and_none_type,
      },
      ["forin"] = {
         before = function(node)
            begin_scope(node)
         end,
         before_statements = function(node)
            local exp1 = node.exps[1]
            local exp1type = resolve_tuple_and_nominal(exp1.type)
            if exp1type.typename == "function" then

               if exp1.op and exp1.op.op == "@funcall" then
                  local t = resolve_tuple_and_nominal(exp1.e2.type)
                  if exp1.e1.tk == "pairs" and is_array_type(t) then
                     node_warning("hint", exp1, "hint: applying pairs on an array: did you intend to apply ipairs?")
                  end

                  if exp1.e1.tk == "pairs" and t.typename ~= "map" then
                     if not (lax and is_unknown(t)) then
                        if is_record_type(t) then
                           match_all_record_field_names(exp1.e2, t, t.field_order,
                           "attempting pairs loop on a record with attributes of different types")
                           local ct = t.typename == "record" and "{string:any}" or "{any:any}"
                           node_warning("hint", exp1.e2, "hint: if you want to iterate over fields of a record, cast it to " .. ct)
                        else
                           node_error(exp1.e2, "cannot apply pairs on values of type: %s", exp1.e2.type)
                        end
                     end
                  elseif exp1.e1.tk == "ipairs" then
                     if t.typename == "tupletable" then
                        local arr_type = arraytype_from_tuple(exp1.e2, t)
                        if not arr_type then
                           node_error(exp1.e2, "attempting ipairs loop on tuple that's not a valid array: %s", exp1.e2.type)
                        end
                     elseif not is_array_type(t) then
                        if not (lax and (is_unknown(t) or t.typename == "emptytable")) then
                           node_error(exp1.e2, "attempting ipairs loop on something that's not an array: %s", exp1.e2.type)
                        end
                     end
                  end
               end

               local last
               local rets = exp1type.rets
               for i, v in ipairs(node.vars) do
                  local r = rets[i]
                  if not r then
                     if rets.is_va then
                        r = last
                     else
                        r = lax and UNKNOWN or INVALID
                     end
                  end
                  add_var(v, v.tk, r)
                  last = r
               end
               if (not lax) and (not rets.is_va and #node.vars > #rets) then
                  local nrets = #rets
                  local at = node.vars[nrets + 1]
                  local n_values = nrets == 1 and "1 value" or tostring(nrets) .. " value%s"
                  node_error(at, "too many variables for this iterator; it produces " .. n_values)
               end
            else
               if not (lax and is_unknown(exp1type)) then
                  node_error(exp1, "expression in for loop does not return an iterator")
               end
            end
         end,
         after = end_scope_and_none_type,
      },
      ["fornum"] = {
         before_statements = function(node, children)
            begin_scope(node)
            local from_t = resolve_tuple_and_nominal(children[2])
            local to_t = resolve_tuple_and_nominal(children[3])
            local step_t = children[4] and resolve_tuple_and_nominal(children[4])
            local t = (from_t.typename == "integer" and
            to_t.typename == "integer" and
            (not step_t or step_t.typename == "integer")) and
            INTEGER or
            NUMBER
            add_var(node.var, node.var.tk, t)
         end,
         after = end_scope_and_none_type,
      },
      ["return"] = {
         after = function(node, children)
            local rets = find_var_type("@return")
            if not rets then

               rets = children[1]
               rets.inferred_at = node
               rets.inferred_at_file = filename
               module_type = resolve_tuple_and_nominal(rets)
               module_type.tk = nil
               st[2]["@return"] = { t = rets }
            end
            local what = "in return value"
            if rets.inferred_at then
               what = what .. inferred_msg(rets)
            end

            local nrets = #rets
            local vatype
            if nrets > 0 then
               vatype = rets.is_va and rets[nrets]
            end

            if #children[1] > nrets and (not lax) and not vatype then
               node_error(node, "in " .. what .. ": excess return values, expected " .. #rets .. " %s, got " .. #children[1] .. " %s", rets, children[1])
            end

            for i = 1, #children[1] do
               local expected = rets[i] or vatype
               if expected then
                  expected = resolve_tuple(expected)
                  local where = (node.exps[i] and node.exps[i].x) and
                  node.exps[i] or
                  node.exps
                  assert(where and where.x)
                  assert_is_a(where, children[1][i], expected, what)
               end
            end

            node.type = NONE
            return node.type
         end,
      },
      ["variable_list"] = {
         after = function(node, children)
            node.type = TUPLE(children)


            local n = #children
            if n > 0 and children[n].typename == "tuple" then
               if children[n].is_va then
                  node.type.is_va = true
               end
               local tuple = children[n]
               for i, c in ipairs(tuple) do
                  children[n + i - 1] = c
               end
            end

            return node.type
         end,
      },
      ["table_literal"] = {
         before = function(node)
            if node.expected then
               if node.expected.typename == "tupletable" then
                  for _, child in ipairs(node) do
                     local n = child.key.constnum
                     if n and is_positive_int(n) then
                        child.value.expected = node.expected.types[n]
                     end
                  end
               elseif is_array_type(node.expected) then
                  for _, child in ipairs(node) do
                     if child.key.constnum then
                        child.value.expected = node.expected.elements
                     end
                  end
               elseif node.expected.typename == "map" then
                  for _, child in ipairs(node) do
                     child.key.expected = node.expected.keys
                     child.value.expected = node.expected.values
                  end
               end

               if is_record_type(node.expected) then
                  for _, child in ipairs(node) do
                     if child.key.conststr then
                        child.value.expected = node.expected.fields[child.key.conststr]
                     end
                  end
               end
            end
         end,
         after = function(node, children)
            node.known = FACT_TRUTHY

            if node.expected then
               local decltype = resolve_tuple_and_nominal(node.expected)

               if decltype.typename == "union" then
                  for _, t in ipairs(decltype.types) do
                     local rt = resolve_tuple_and_nominal(t)
                     if is_known_table_type(rt) then
                        node.expected = t
                        decltype = rt
                        break
                     end
                  end
                  if decltype.typename == "union" then
                     node_error(node, "unexpected table literal, expected: %s", decltype)
                  end
               end

               if not is_known_table_type(decltype) then
                  node.type = infer_table_literal(node, children)
                  return node.type
               end

               local is_record = is_record_type(decltype)
               local is_array = is_array_type(decltype)
               local is_tupletable = decltype.typename == "tupletable"
               local is_map = decltype.typename == "map"

               local force_array = nil

               local seen_keys = {}

               for i, child in ipairs(children) do
                  assert(child.typename == "table_item")
                  local cvtype = resolve_tuple(child.vtype)
                  local ck = child.kname
                  local n = node[i].key.constnum
                  check_redeclared_key(node.expected_context, node[i], seen_keys, ck, n)
                  if is_record and ck then
                     local df = decltype.fields[ck]
                     if not df then
                        node_error(node[i], in_context(node.expected_context, "unknown field " .. ck))
                     else
                        assert_is_a(node[i], cvtype, df, "in record field", ck)
                     end
                  elseif is_tupletable and is_number_type(child.ktype) then
                     local dt = decltype.types[n]
                     if not n then
                        node_error(node[i], in_context(node.expected_context, "unknown index in tuple %s"), decltype)
                     elseif not dt then
                        node_error(node[i], in_context(node.expected_context, "unexpected index " .. n .. " in tuple %s"), decltype)
                     else
                        assert_is_a(node[i], cvtype, dt, in_context(node.expected_context, "in tuple"), "at index " .. tostring(n))
                     end
                  elseif is_array and is_number_type(child.ktype) then
                     if child.vtype.typename == "tuple" and i == #children and node[i].key_parsed == "implicit" then

                        for ti, tt in ipairs(child.vtype) do
                           assert_is_a(node[i], tt, decltype.elements, in_context(node.expected_context, "expected an array"), "at index " .. tostring(i + ti - 1))
                        end
                     else
                        assert_is_a(node[i], cvtype, decltype.elements, in_context(node.expected_context, "expected an array"), "at index " .. tostring(n))
                     end
                  elseif node[i].key_parsed == "implicit" then
                     force_array = expand_type(node[i], force_array, child.vtype)
                  elseif is_map then
                     assert_is_a(node[i], child.ktype, decltype.keys, in_context(node.expected_context, "in map key"))
                     assert_is_a(node[i], cvtype, decltype.values, in_context(node.expected_context, "in map value"))
                  else
                     node_error(node[i], in_context(node.expected_context, "unexpected key of type %s in table of type %s"), child.ktype, decltype)
                  end
               end

               if force_array then
                  node.type = a_type({
                     inferred_at = node,
                     inferred_at_file = filename,
                     typename = "array",
                     elements = force_array,
                  })
               else
                  node.type = resolve_typevars_at(node.expected, node)
               end
            else
               node.type = infer_table_literal(node, children)
            end

            return node.type
         end,
      },
      ["table_item"] = {
         after = function(node, children)
            local kname = node.key.conststr
            local ktype = children[1]
            local vtype = children[2]
            if node.decltype then
               vtype = node.decltype
               assert_is_a(node.value, children[2], node.decltype, "in table item")
            end
            node.type = a_type({
               y = node.y,
               x = node.x,
               typename = "table_item",
               kname = kname,
               ktype = ktype,
               vtype = vtype,
            })
            return node.type
         end,
      },
      ["local_function"] = {
         before = function(node)
            reserve_symbol_list_slot(node)
            begin_scope(node)
         end,
         before_statements = function(node)
            add_internal_function_variables(node)
            add_function_definition_for_recursion(node)
         end,
         after = function(node, children)
            end_function_scope(node)
            local rets = get_rets(children[3])

            add_var(node, node.name.tk, a_type({
               y = node.y,
               x = node.x,
               typename = "function",
               typeargs = node.typeargs,
               args = children[2],
               rets = rets,
               filename = filename,
            }))
            return node.type
         end,
      },
      ["global_function"] = {
         before = function(node)
            begin_scope(node)
         end,
         before_statements = function(node)
            add_internal_function_variables(node)
            add_function_definition_for_recursion(node)
         end,
         after = function(node, children)
            end_function_scope(node)
            add_global(node, node.name.tk, a_type({
               y = node.y,
               x = node.x,
               typename = "function",
               typeargs = node.typeargs,
               args = children[2],
               rets = get_rets(children[3]),
               filename = filename,
            }))
            return node.type
         end,
      },
      ["record_function"] = {
         before = function(node)
            begin_scope(node)
         end,
         before_statements = function(node, children)
            add_internal_function_variables(node)

            local rtype = resolve_tuple_and_nominal(resolve_typetype(children[1]))
            local owner = find_record_to_extend(node.fn_owner)

            if node.is_method then
               children[3][1] = rtype
               add_var(nil, "self", rtype)
            end

            if rtype.typename == "emptytable" then
               rtype.typename = "record"
               rtype.fields = {}
               rtype.field_order = {}
            end
            if is_record_type(rtype) then
               local fn_type = a_type({
                  y = node.y,
                  x = node.x,
                  typename = "function",
                  is_method = node.is_method,
                  typeargs = node.typeargs,
                  args = children[3],
                  rets = get_rets(children[4]),
                  filename = filename,
               })

               local ok = false
               if lax then
                  ok = true
               elseif rtype.fields[node.name.tk] and is_a(fn_type, rtype.fields[node.name.tk]) then
                  ok = true
               elseif owner == rtype then
                  ok = true
               end

               if ok then
                  rtype.fields[node.name.tk] = fn_type
                  table.insert(rtype.field_order, node.name.tk)
                  node.name.type = fn_type
               else
                  local name = tl.pretty_print_ast(node.fn_owner, { preserve_indent = true, preserve_newlines = false })
                  node_error(node, "cannot add undeclared function '" .. node.name.tk .. "' outside of the scope where '" .. name .. "' was originally declared")
               end
            else
               if not (lax and rtype.typename == "unknown") then
                  node_error(node, "not a module: %s", rtype)
               end
            end
         end,
         after = function(node, _children)
            end_function_scope(node)
            node.type = NONE
            return node.type
         end,
      },
      ["function"] = {
         before = function(node)
            begin_scope(node)
         end,
         before_statements = function(node)
            add_internal_function_variables(node)
         end,
         after = function(node, children)
            end_function_scope(node)


            node.type = a_type({
               y = node.y,
               x = node.x,
               typename = "function",
               typeargs = node.typeargs,
               args = children[1],
               rets = children[2],
               filename = filename,
            })
            return node.type
         end,
      },
      ["cast"] = {
         after = function(node, _children)
            node.type = node.casttype
            return node.type
         end,
      },
      ["paren"] = {
         after = function(node, children)
            node.known = node.e1 and node.e1.known
            node.type = resolve_tuple(children[1])
            return node.type
         end,
      },
      ["op"] = {
         before = function()
            begin_scope()
         end,
         before_e2 = function(node)
            if node.op.op == "and" then
               apply_facts(node, node.e1.known)
            elseif node.op.op == "or" then
               apply_facts(node, facts_not(node.e1.known, node))
            elseif node.op.op == "@funcall" then
               if node.e1.type.typename == "function" then
                  local argdelta = (node.e1.op and node.e1.op.op == ":") and -1 or 0
                  for i, typ in ipairs(node.e1.type.args) do
                     if node.e2[i + argdelta] then
                        node.e2[i + argdelta].expected = typ
                     end
                  end
               end
               apply_facts(node, facts_not(node.e1.known, node))
            elseif node.op.op == "@index" then
               if node.e1.type.typename == "map" then
                  node.e2.expected = node.e1.type.keys
               end
            end
         end,
         after = function(node, children)
            end_scope()

            local a = children[1]
            local b = children[3]

            local orig_a = a
            local orig_b = b
            local ra = a and resolve_tuple_and_nominal(a)
            local rb = b and resolve_tuple_and_nominal(b)
            if ra and is_typetype(ra) and ra.def.typename == "record" then
               ra = ra.def
            end
            if rb and is_typetype(rb) and rb.def.typename == "record" then
               rb = rb.def
            end
            if node.op.op == "." then
               a = ra
               if a.typename == "map" then
                  if is_a(a.keys, STRING) or is_a(a.keys, ANY) then
                     node.type = a.values
                  else
                     node_error(node, "cannot use . index, expects keys of type %s", a.keys)
                  end
               else
                  node.type = match_record_key(node, a, { y = node.e2.y, x = node.e2.x, kind = "string", tk = node.e2.tk }, orig_a)
                  if node.type.needs_compat and opts.gen_compat ~= "off" then

                     if node.e1.kind == "variable" and node.e2.kind == "identifier" then
                        local key = node.e1.tk .. "." .. node.e2.tk
                        node.kind = "variable"
                        node.tk = "_tl_" .. node.e1.tk .. "_" .. node.e2.tk
                        all_needs_compat[key] = true
                     end
                  end
               end
            elseif node.op.op == "@funcall" then
               node.type = type_check_funcall(node, a, b)
            elseif node.op.op == "@index" then
               node.type = type_check_index(node, node.e2, a, b)
            elseif node.op.op == "as" then
               node.type = b
            elseif node.op.op == "is" then
               if ra.typename == "typetype" then
                  node_error(node, "can only use 'is' on variables, not types")
               elseif node.e1.kind == "variable" then
                  node.known = Fact({ fact = "is", var = node.e1.tk, typ = b, where = node })
               else
                  node_error(node, "can only use 'is' on variables")
               end
               node.type = BOOLEAN
            elseif node.op.op == ":" then
               node.type = match_record_key(node, node.e1.type, node.e2, orig_a)
            elseif node.op.op == "not" then
               node.known = facts_not(node.e1.known, node)
               node.type = BOOLEAN
            elseif node.op.op == "and" then
               node.known = facts_and(node.e1.known, node.e2.known, node)
               node.type = resolve_tuple(b)
            elseif node.op.op == "or" and is_known_table_type(ra) and b.typename == "emptytable" then
               node.known = nil
               node.type = resolve_tuple(a)
            elseif node.op.op == "or" and is_a(rb, ra) then
               node.known = facts_or(node.e1.known, node.e2.known)
               node.type = resolve_tuple(a)
            elseif node.op.op == "or" and b.typename == "nil" then
               node.known = nil
               node.type = resolve_tuple(a)
            elseif node.op.op == "or" and
               ((ra.typename == "enum" and rb.typename == "string" and is_a(rb, ra)) or
               (ra.typename == "string" and rb.typename == "enum" and is_a(ra, rb))) then
               node.known = nil
               node.type = (ra.typename == "enum" and ra or rb)
            elseif node.op.op == "or" and node.expected and node.expected.typename == "union" then

               node.known = facts_or(node.e1.known, node.e2.known)
               local u = unite({ ra, rb }, true)
               local valid, err = is_valid_union(u)
               node.type = valid and u or node_error(node, err)
            elseif node.op.op == "==" or node.op.op == "~=" then
               node.type = BOOLEAN
               if is_a(b, a, true) or a.typename == "typevar" then
                  if node.op.op == "==" and node.e1.kind == "variable" then
                     node.known = Fact({ fact = "==", var = node.e1.tk, typ = b, where = node })
                  end
               elseif is_a(a, b, true) or b.typename == "typevar" then
                  if node.op.op == "==" and node.e2.kind == "variable" then
                     node.known = Fact({ fact = "==", var = node.e2.tk, typ = a, where = node })
                  end
               elseif lax and (is_unknown(a) or is_unknown(b)) then
                  node.type = UNKNOWN
               else
                  node_error(node, "types are not comparable for equality: %s and %s", a, b)
               end
            elseif node.op.arity == 1 and unop_types[node.op.op] then
               a = ra
               if a.typename == "union" then
                  a = unite(a.types, true)
               end

               local types_op = unop_types[node.op.op]
               node.type = types_op[a.typename]
               local metamethod
               if node.type then
                  if node.type.typename ~= "boolean" then
                     node.known = FACT_TRUTHY
                  end
               else
                  metamethod = a.meta_fields and a.meta_fields[unop_to_metamethod[node.op.op] or ""]
                  if metamethod then
                     node.type = resolve_tuple_and_nominal(type_check_function_call(node, metamethod, { a }, false, 0))
                  elseif lax and is_unknown(a) then
                     node.type = UNKNOWN
                  else
                     node_error(node, "cannot use operator '" .. node.op.op:gsub("%%", "%%%%") .. "' on type %s", resolve_tuple(orig_a))
                  end
               end

               if node.op.op == "~" and env.gen_target == "5.1" then
                  if metamethod then
                     all_needs_compat["mt"] = true
                     convert_node_to_compat_mt_call(node, unop_to_metamethod[node.op.op], 1, node.e1)
                  else
                     all_needs_compat["bit32"] = true
                     convert_node_to_compat_call(node, "bit32", "bnot", node.e1)
                  end
               end

            elseif node.op.arity == 2 and binop_types[node.op.op] then
               if node.op.op == "or" then
                  node.known = facts_or(node.e1.known, node.e2.known)
               end

               a = ra
               b = rb

               if a.typename == "union" then
                  a = unite(a.types, true)
               end
               if b.typename == "union" then
                  b = unite(b.types, true)
               end

               local types_op = binop_types[node.op.op]
               node.type = types_op[a.typename] and types_op[a.typename][b.typename]
               local metamethod
               local meta_self = 1
               if node.type then
                  if types_op == numeric_binop or node.op.op == ".." then
                     node.known = FACT_TRUTHY
                  end
               else
                  metamethod = a.meta_fields and a.meta_fields[binop_to_metamethod[node.op.op] or ""]
                  if not metamethod then
                     metamethod = b.meta_fields and b.meta_fields[binop_to_metamethod[node.op.op] or ""]
                     meta_self = 2
                  end
                  if metamethod then
                     node.type = resolve_tuple_and_nominal(type_check_function_call(node, metamethod, { a, b }, false, 0))
                  elseif lax and (is_unknown(a) or is_unknown(b)) then
                     node.type = UNKNOWN
                  else
                     node_error(node, "cannot use operator '" .. node.op.op:gsub("%%", "%%%%") .. "' for types %s and %s", resolve_tuple(orig_a), resolve_tuple(orig_b))
                  end
               end

               if node.op.op == "//" and env.gen_target == "5.1" then
                  if metamethod then
                     all_needs_compat["mt"] = true
                     convert_node_to_compat_mt_call(node, "__idiv", meta_self, node.e1, node.e2)
                  else
                     local div = { y = node.y, x = node.x, kind = "op", op = an_operator(node, 2, "/"), e1 = node.e1, e2 = node.e2 }
                     convert_node_to_compat_call(node, "math", "floor", div)
                  end
               elseif bit_operators[node.op.op] and env.gen_target == "5.1" then
                  if metamethod then
                     all_needs_compat["mt"] = true
                     convert_node_to_compat_mt_call(node, binop_to_metamethod[node.op.op], meta_self, node.e1, node.e2)
                  else
                     all_needs_compat["bit32"] = true
                     convert_node_to_compat_call(node, "bit32", bit_operators[node.op.op], node.e1, node.e2)
                  end
               end
            else
               error("unknown node op " .. node.op.op)
            end
            return node.type
         end,
      },
      ["variable"] = {
         after = function(node, _children)
            if node.tk == "..." then
               local va_sentinel = find_var_type("@is_va")
               if not va_sentinel or va_sentinel.typename == "nil" then
                  node_error(node, "cannot use '...' outside a vararg function")
               end
            end

            if node.tk == "_G" then
               node.type, node.is_const = simulate_g()
            else
               node.type, node.is_const = find_var_type(node.tk)
            end
            if node.type and is_typetype(node.type) then
               node.type = a_type({
                  y = node.y,
                  x = node.x,
                  typename = "nominal",
                  names = { node.tk },
                  found = node.type,
                  resolved = node.type,
               })
            end
            if node.type == nil then
               node.type = a_type({ typename = "unknown" })
               if lax then
                  add_unknown(node, node.tk)
               else
                  node_error(node, "unknown variable: " .. node.tk)
               end
            end
            return node.type
         end,
      },
      ["type_identifier"] = {
         after = function(node, _children)
            node.type, node.is_const = find_var_type(node.tk)
            if node.type == nil then
               if lax then
                  node.type = UNKNOWN
                  add_unknown(node, node.tk)
               else
                  node_error(node, "unknown variable: " .. node.tk)
               end
            end
            return node.type
         end,
      },
      ["argument"] = {
         after = function(node, _children)
            local t = node.decltype
            if not t then
               t = UNKNOWN
            end
            if node.tk == "..." then
               t = a_type({ typename = "tuple", is_va = true, t })
            end
            add_var(node, node.tk, t).is_func_arg = true
            return node.type
         end,
      },
      ["identifier"] = {
         after = function(node, _children)
            node.type = node.type or NONE
            return node.type
         end,
      },
      ["newtype"] = {
         after = function(node, _children)
            node.type = node.type or node.newtype
            return node.type
         end,
      },
      ["error_node"] = {
         after = function(node, _children)
            node.type = INVALID
            return node.type
         end,
      },
   }

   visit_node.cbs["string"] = {
      after = function(node, _children)
         node.type = a_type({
            y = node.y,
            x = node.x,
            typename = node.kind,
            tk = node.tk,
         })
         node.known = FACT_TRUTHY
         return node.type
      end,
   }
   visit_node.cbs["number"] = visit_node.cbs["string"]
   visit_node.cbs["integer"] = visit_node.cbs["string"]
   visit_node.cbs["boolean"] = {
      after = function(node, _children)
         node.type = a_type({
            y = node.y,
            x = node.x,
            typename = node.kind,
            tk = node.tk,
         })
         if node.tk == "true" then
            node.known = FACT_TRUTHY
         end
         return node.type
      end,
   }
   visit_node.cbs["nil"] = visit_node.cbs["boolean"]

   visit_node.cbs["do"] = visit_node.cbs["if"]
   visit_node.cbs["..."] = visit_node.cbs["variable"]
   visit_node.cbs["break"] = visit_node.cbs["if"]
   visit_node.cbs["argument_list"] = visit_node.cbs["variable_list"]
   visit_node.cbs["expression_list"] = visit_node.cbs["variable_list"]

   visit_node.after = function(node, _children)
      if type(node.type) ~= "table" then
         error(node.kind .. " did not produce a type")
      end
      if type(node.type.typename) ~= "string" then
         error(node.kind .. " type does not have a typename")
      end
      return node.type
   end

   local visit_type = {
      cbs = {
         ["string"] = {
            after = function(typ, _children)
               return typ
            end,
         },
         ["function"] = {
            before = function(_typ, _children)
               begin_scope()
            end,
            after = function(typ, _children)
               end_scope()
               return typ
            end,
         },
         ["record"] = {
            before = function(typ, _children)
               begin_scope()
               for name, typ2 in fields_of(typ) do
                  if typ2.typename == "typetype" then
                     typ2.typename = "nestedtype"
                     local resolved, is_alias = resolve_nominal_typetype(typ2)
                     if is_alias then
                        typ2.is_alias = true
                        typ2.def.resolved = resolved
                     end
                     add_var(nil, name, resolved)
                  end
               end
            end,
            after = function(typ, _children)
               end_scope()
               for _, typ2 in fields_of(typ) do
                  if typ2.typename == "nestedtype" then
                     typ2.typename = "typetype"
                  end
               end
               return typ
            end,
         },
         ["typearg"] = {
            after = function(typ, _children)
               add_var(nil, typ.typearg, a_type({
                  y = typ.y,
                  x = typ.x,
                  typename = "typearg",
                  typearg = typ.typearg,
               }))
               return typ
            end,
         },
         ["typevar"] = {
            after = function(typ, _children)
               if not find_var_type(typ.typevar) then
                  type_error(typ, "undefined type variable " .. typ.typevar)
               end
               return typ
            end,
         },
         ["nominal"] = {
            after = function(typ, _children)
               if typ.found then
                  return typ
               end

               local t = find_type(typ.names, true)
               if t then
                  if t.typename == "typearg" then

                     typ.names = nil
                     typ.typename = "typevar"
                     typ.typevar = t.typearg
                  else
                     typ.found = t
                  end
               else
                  local name = typ.names[1]
                  local unresolved = find_var_type("@unresolved")
                  if not unresolved then
                     unresolved = { typename = "unresolved", labels = {}, nominals = {} }
                     add_var(nil, "@unresolved", unresolved)
                  end
                  unresolved.nominals[name] = unresolved.nominals[name] or {}
                  table.insert(unresolved.nominals[name], typ)
               end
               return typ
            end,
         },
         ["union"] = {
            after = function(typ, _children)
               local valid, err = is_valid_union(typ)
               if not valid then
                  type_error(typ, err, typ)
               end
               return typ
            end,
         },
      },
      after = function(typ, _children, ret)
         if type(ret) ~= "table" then
            error(typ.typename .. " did not produce a type")
         end
         if type(ret.typename) ~= "string" then
            error("type node does not have a typename")
         end
         return ret
      end,
   }

   if not opts.run_internal_compiler_checks then
      visit_node.after = nil
      visit_type.after = nil
   end

   visit_type.cbs["tupletable"] = visit_type.cbs["string"]
   visit_type.cbs["typetype"] = visit_type.cbs["string"]
   visit_type.cbs["nestedtype"] = visit_type.cbs["string"]
   visit_type.cbs["array"] = visit_type.cbs["string"]
   visit_type.cbs["map"] = visit_type.cbs["string"]
   visit_type.cbs["arrayrecord"] = visit_type.cbs["record"]
   visit_type.cbs["enum"] = visit_type.cbs["string"]
   visit_type.cbs["boolean"] = visit_type.cbs["string"]
   visit_type.cbs["nil"] = visit_type.cbs["string"]
   visit_type.cbs["number"] = visit_type.cbs["string"]
   visit_type.cbs["integer"] = visit_type.cbs["string"]
   visit_type.cbs["thread"] = visit_type.cbs["string"]
   visit_type.cbs["bad_nominal"] = visit_type.cbs["string"]
   visit_type.cbs["emptytable"] = visit_type.cbs["string"]
   visit_type.cbs["table_item"] = visit_type.cbs["string"]
   visit_type.cbs["unresolved_emptytable_value"] = visit_type.cbs["string"]
   visit_type.cbs["tuple"] = visit_type.cbs["string"]
   visit_type.cbs["poly"] = visit_type.cbs["string"]
   visit_type.cbs["any"] = visit_type.cbs["string"]
   visit_type.cbs["unknown"] = visit_type.cbs["string"]
   visit_type.cbs["invalid"] = visit_type.cbs["string"]
   visit_type.cbs["unresolved"] = visit_type.cbs["string"]
   visit_type.cbs["none"] = visit_type.cbs["string"]

   assert(ast.kind == "statements")
   recurse_node(ast, visit_node, visit_type)

   close_types(st[1])
   check_for_unused_vars(st[1])

   clear_redundant_errors(errors)

   add_compat_entries(ast, all_needs_compat, env.gen_compat)

   local result = {
      ast = ast,
      env = env,
      type = module_type,
      filename = filename,
      warnings = warnings,
      type_errors = errors,
      symbol_list = symbol_list,
      dependencies = dependencies,
   }

   env.loaded[filename] = result
   table.insert(env.loaded_order, filename)

   return result
end

local typename_to_typecode = {
   ["typevar"] = tl.typecodes.TYPE_VARIABLE,
   ["typearg"] = tl.typecodes.TYPE_VARIABLE,
   ["function"] = tl.typecodes.FUNCTION,
   ["array"] = tl.typecodes.ARRAY,
   ["map"] = tl.typecodes.MAP,
   ["tupletable"] = tl.typecodes.TUPLE,
   ["arrayrecord"] = tl.typecodes.ARRAYRECORD,
   ["record"] = tl.typecodes.RECORD,
   ["enum"] = tl.typecodes.ENUM,
   ["boolean"] = tl.typecodes.BOOLEAN,
   ["string"] = tl.typecodes.STRING,
   ["nil"] = tl.typecodes.NIL,
   ["thread"] = tl.typecodes.THREAD,
   ["number"] = tl.typecodes.NUMBER,
   ["integer"] = tl.typecodes.INTEGER,
   ["union"] = tl.typecodes.IS_UNION,
   ["nominal"] = tl.typecodes.NOMINAL,
   ["emptytable"] = tl.typecodes.EMPTY_TABLE,
   ["unresolved_emptytable_value"] = tl.typecodes.EMPTY_TABLE,
   ["poly"] = tl.typecodes.IS_POLY,
   ["any"] = tl.typecodes.ANY,
   ["unknown"] = tl.typecodes.UNKNOWN,
   ["invalid"] = tl.typecodes.INVALID,
}

function tl.get_types(result, trenv)
   local filename = result.filename or "?"

   local function mark_array(x)
      local arr = x
      arr[0] = false
      return x
   end

   if not trenv then
      trenv = {
         next_num = 1,
         typeid_to_num = {},
         tr = {
            by_pos = {},
            types = {},
            symbols = mark_array({}),
            globals = {},
         },
      }
   end

   local tr = trenv.tr
   local typeid_to_num = trenv.typeid_to_num

   local get_typenum

   local function store_function(ti, rt)
      local args = {}
      for _, fnarg in ipairs(rt.args) do
         table.insert(args, mark_array({ get_typenum(fnarg), nil }))
      end
      ti.args = mark_array(args)
      local rets = {}
      for _, fnarg in ipairs(rt.rets) do
         table.insert(rets, mark_array({ get_typenum(fnarg), nil }))
      end
      ti.rets = mark_array(rets)
      ti.vararg = not not rt.is_va
   end

   get_typenum = function(t)
      assert(t.typeid)

      local n = typeid_to_num[t.typeid]
      if n then
         return n
      end


      n = trenv.next_num

      local rt = t
      if rt.typename == "typetype" or rt.typename == "nestedtype" then
         rt = rt.def
      elseif rt.typename == "tuple" and #rt == 1 then
         rt = rt[1]
      end

      local ti = {
         t = assert(typename_to_typecode[rt.typename]),
         str = show_type(t, true),
         file = t.filename,
         y = t.y,
         x = t.x,
      }
      tr.types[n] = ti
      typeid_to_num[t.typeid] = n
      trenv.next_num = trenv.next_num + 1

      if t.found then
         ti.ref = get_typenum(t.found)
      end
      if t.resolved then
         rt = t
      end
      assert(rt.typename ~= "typetype")

      if is_record_type(rt) then

         local r = {}
         for _, k in ipairs(rt.field_order) do
            local v = rt.fields[k]
            r[k] = get_typenum(v)
         end
         ti.fields = r
      end

      if is_array_type(rt) then
         ti.elements = get_typenum(rt.elements)
      end

      if rt.typename == "map" then
         ti.keys = get_typenum(rt.keys)
         ti.values = get_typenum(rt.values)
      elseif rt.typename == "enum" then
         ti.enums = mark_array(sorted_keys(rt.enumset))
      elseif rt.typename == "function" then
         store_function(ti, rt)
      elseif rt.typename == "poly" or rt.typename == "union" or rt.typename == "tupletable" then
         local tis = {}

         for _, pt in ipairs(rt.types) do
            table.insert(tis, get_typenum(pt))
         end

         ti.types = mark_array(tis)
      end

      return n
   end

   local visit_node = { allow_missing_cbs = true }
   local visit_type = { allow_missing_cbs = true }

   local skip = {
      ["none"] = true,
      ["tuple"] = true,
      ["table_item"] = true,
   }

   local ft = {}
   tr.by_pos[filename] = ft

   local function store(y, x, typ)
      if not typ or skip[typ.typename] then
         return
      end

      local yt = ft[y]
      if not yt then
         yt = {}
         ft[y] = yt
      end

      yt[x] = get_typenum(typ)
   end

   visit_node.after = function(node)
      store(node.y, node.x, node.type)
   end

   visit_type.after = function(typ)
      store(typ.y or 0, typ.x or 0, typ)
   end

   recurse_node(result.ast, visit_node, visit_type)

   tr.by_pos[filename][0] = nil


   do
      local n = 0
      local p = 0
      local n_stack, p_stack = {}, {}
      local level = 0
      for i, s in ipairs(result.symbol_list) do
         if s.typ then
            n = n + 1
         elseif s.name == "@{" then
            level = level + 1
            n_stack[level], p_stack[level] = n, p
            n, p = 0, i
         else
            if n == 0 then
               result.symbol_list[p].skip = true
               s.skip = true
            end
            n, p = n_stack[level], p_stack[level]
            level = level - 1
         end
      end
   end


   do
      local stack = {}
      local level = 0
      local i = 0
      for _, s in ipairs(result.symbol_list) do
         if not s.skip then
            i = i + 1
            local id
            if s.typ then
               id = get_typenum(s.typ)
            elseif s.name == "@{" then
               level = level + 1
               stack[level] = i
               id = -1
            else
               local other = stack[level]
               level = level - 1
               tr.symbols[other][4] = i
               id = other - 1
            end
            local sym = mark_array({ s.y, s.x, s.name, id })
            table.insert(tr.symbols, sym)
         end
      end
   end

   local gkeys = sorted_keys(result.env.globals)
   for _, name in ipairs(gkeys) do
      if name:sub(1, 1) ~= "@" then
         local var = result.env.globals[name]
         tr.globals[name] = get_typenum(var.t)
      end
   end

   return tr, trenv
end

function tl.symbols_in_scope(tr, y, x)
   local function find(symbols, at_y, at_x)
      local function le(a, b)
         return a[1] < b[1] or
         (a[1] == b[1] and a[2] <= b[2])
      end
      return binary_search(symbols, { at_y, at_x }, le) or 0
   end

   local ret = {}








   local n = find(tr.symbols, y, x)

   local symbols = tr.symbols
   while n >= 1 do
      local s = symbols[n]
      if s[3] == "@{" then
         n = n - 1
      elseif s[3] == "@}" then
         n = s[4]
      else
         ret[s[3]] = s[4]
         n = n - 1
      end
   end

   return ret
end

tl.process = function(filename, env)
   if env and env.loaded and env.loaded[filename] then
      return env.loaded[filename]
   end
   local fd, err = io.open(filename, "r")
   if not fd then
      return nil, "could not open " .. filename .. ": " .. err
   end

   local input; input, err = fd:read("*a")
   fd:close()
   if not input then
      return nil, "could not read " .. filename .. ": " .. err
   end

   local _, extension = filename:match("(.*)%.([a-z]+)$")
   extension = extension and extension:lower()

   local is_lua
   if extension == "tl" then
      is_lua = false
   elseif extension == "lua" then
      is_lua = true
   else
      is_lua = input:match("^#![^\n]*lua[^\n]*\n")
   end

   return tl.process_string(input, is_lua, env, filename)
end

function tl.process_string(input, is_lua, env, filename)

   env = env or tl.init_env(is_lua)
   if env.loaded and env.loaded[filename] then
      return env.loaded[filename]
   end
   filename = filename or ""

   local syntax_errors = {}
   local tokens, errs = tl.lex(input)
   if errs then
      for _, err in ipairs(errs) do
         table.insert(syntax_errors, {
            y = err.y,
            x = err.x,
            msg = "invalid token '" .. err.tk .. "'",
            filename = filename,
         })
      end
   end

   local _, program = tl.parse_program(tokens, syntax_errors, filename)

   if (not env.keep_going) and #syntax_errors > 0 then
      local result = {
         ok = false,
         filename = filename,
         type_errors = {},
         syntax_errors = syntax_errors,
         env = env,
      }
      env.loaded[filename] = result
      table.insert(env.loaded_order, filename)
      return result
   end

   local opts = {
      filename = filename,
      lax = is_lua,
      gen_compat = env.gen_compat,
      env = env,
   }
   local result = tl.type_check(program, opts)

   result.syntax_errors = syntax_errors

   return result
end

tl.gen = function(input, env)
   env = env or tl.init_env()
   local result = tl.process_string(input, false, env)

   if (not result.ast) or #result.syntax_errors > 0 then
      return nil, result
   end

   return tl.pretty_print_ast(result.ast), result
end

local function tl_package_loader(module_name)
   local found_filename, fd, tried = tl.search_module(module_name, false)
   if found_filename then
      local input = fd:read("*a")
      if not input then
         return table.concat(tried, "\n\t")
      end
      fd:close()
      local errs = {}
      local _, program = tl.parse_program(tl.lex(input), errs, module_name)
      if #errs > 0 then
         error(found_filename .. ":" .. errs[1].y .. ":" .. errs[1].x .. ": " .. errs[1].msg)
      end
      local lax = not not found_filename:match("lua$")
      if not tl.package_loader_env then
         tl.package_loader_env = tl.init_env(lax)
      end

      tl.type_check(program, {
         lax = lax,
         filename = found_filename,
         env = tl.package_loader_env,
         run_internal_compiler_checks = false,
      })

      local code = tl.pretty_print_ast(program, true)
      local chunk, err = load(code, module_name, "t")
      if chunk then
         return function()
            local ret = chunk()
            package.loaded[module_name] = ret
            return ret
         end
      else
         error("Internal Compiler Error: Teal generator produced invalid Lua. Please report a bug at https://github.com/teal-language/tl\n\n" .. err)
      end
   end
   return table.concat(tried, "\n\t")
end

function tl.loader()
   if package.searchers then
      table.insert(package.searchers, 2, tl_package_loader)
   else
      table.insert(package.loaders, 2, tl_package_loader)
   end
end

tl.load = function(input, chunkname, mode, env)
   local tokens = tl.lex(input)
   local errs = {}
   local _, program = tl.parse_program(tokens, errs, chunkname)
   if #errs > 0 then
      return nil, (chunkname or "") .. ":" .. errs[1].y .. ":" .. errs[1].x .. ": " .. errs[1].msg
   end
   local code = tl.pretty_print_ast(program, true)
   return load(code, chunkname, mode, env)
end

return tl