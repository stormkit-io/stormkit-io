import path from "node:path";
import http, { ServerResponse } from "node:http";
import stream, { Readable } from "node:stream";
import stream$1 from "stream";
import fs from "node:fs";
var extendStatics = function(d, b) {
  extendStatics = Object.setPrototypeOf || { __proto__: [] } instanceof Array && function(d2, b2) {
    d2.__proto__ = b2;
  } || function(d2, b2) {
    for (var p in b2) if (Object.prototype.hasOwnProperty.call(b2, p)) d2[p] = b2[p];
  };
  return extendStatics(d, b);
};
function __extends(d, b) {
  if (typeof b !== "function" && b !== null)
    throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
  extendStatics(d, b);
  function __() {
    this.constructor = d;
  }
  d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
}
var __assign = function() {
  __assign = Object.assign || function __assign2(t) {
    for (var s, i = 1, n = arguments.length; i < n; i++) {
      s = arguments[i];
      for (var p in s) if (Object.prototype.hasOwnProperty.call(s, p)) t[p] = s[p];
    }
    return t;
  };
  return __assign.apply(this, arguments);
};
function __awaiter(thisArg, _arguments, P, generator) {
  function adopt(value) {
    return value instanceof P ? value : new P(function(resolve) {
      resolve(value);
    });
  }
  return new (P || (P = Promise))(function(resolve, reject) {
    function fulfilled(value) {
      try {
        step(generator.next(value));
      } catch (e) {
        reject(e);
      }
    }
    function rejected(value) {
      try {
        step(generator["throw"](value));
      } catch (e) {
        reject(e);
      }
    }
    function step(result) {
      result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected);
    }
    step((generator = generator.apply(thisArg, [])).next());
  });
}
function __generator(thisArg, body) {
  var _ = { label: 0, sent: function() {
    if (t[0] & 1) throw t[1];
    return t[1];
  }, trys: [], ops: [] }, f, y, t, g = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
  return g.next = verb(0), g["throw"] = verb(1), g["return"] = verb(2), typeof Symbol === "function" && (g[Symbol.iterator] = function() {
    return this;
  }), g;
  function verb(n) {
    return function(v) {
      return step([n, v]);
    };
  }
  function step(op) {
    if (f) throw new TypeError("Generator is already executing.");
    while (g && (g = 0, op[0] && (_ = 0)), _) try {
      if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
      if (y = 0, t) op = [op[0] & 2, t.value];
      switch (op[0]) {
        case 0:
        case 1:
          t = op;
          break;
        case 4:
          _.label++;
          return { value: op[1], done: false };
        case 5:
          _.label++;
          y = op[1];
          op = [0];
          continue;
        case 7:
          op = _.ops.pop();
          _.trys.pop();
          continue;
        default:
          if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) {
            _ = 0;
            continue;
          }
          if (op[0] === 3 && (!t || op[1] > t[0] && op[1] < t[3])) {
            _.label = op[1];
            break;
          }
          if (op[0] === 6 && _.label < t[1]) {
            _.label = t[1];
            t = op;
            break;
          }
          if (t && _.label < t[2]) {
            _.label = t[2];
            _.ops.push(op);
            break;
          }
          if (t[2]) _.ops.pop();
          _.trys.pop();
          continue;
      }
      op = body.call(thisArg, _);
    } catch (e) {
      op = [6, e];
      y = 0;
    } finally {
      f = t = 0;
    }
    if (op[0] & 5) throw op[1];
    return { value: op[0] ? op[1] : void 0, done: true };
  }
}
function __spreadArray(to, from, pack) {
  if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
    if (ar || !(i in from)) {
      if (!ar) ar = Array.prototype.slice.call(from, 0, i);
      ar[i] = from[i];
    }
  }
  return to.concat(ar || Array.prototype.slice.call(from));
}
typeof SuppressedError === "function" ? SuppressedError : function(error, suppressed, message) {
  var e = new Error(message);
  return e.name = "SuppressedError", e.error = error, e.suppressed = suppressed, e;
};
var originalConsole = console;
var originalStdout = process.stdout.write;
var originalStderr = process.stderr.write;
var Logger = (
  /** @class */
  function() {
    function Logger2() {
      this.stdout = [];
      var stdout = this.streamWithContext("info");
      var stderr = this.streamWithContext("error");
      process.stdout.write = stdout.write.bind(stdout);
      process.stderr.write = stderr.write.bind(stdout);
      console = new console.Console(this.streamWithContext("info"), this.streamWithContext("error"));
    }
    Logger2.prototype.streamWithContext = function(level) {
      var _this = this;
      return new stream.Writable({
        write: function(chunk, _, callback) {
          _this.stdout.push({
            ts: Date.now(),
            msg: chunk.toString(),
            level
          });
          callback(null);
        }
      });
    };
    Logger2.prototype.logs = function() {
      this.restore();
      return this.stdout;
    };
    Logger2.prototype.restore = function() {
      console = originalConsole;
      process.stdout.write = originalStdout;
      process.stderr.write = originalStderr;
    };
    return Logger2;
  }()
);
var Request = (
  /** @class */
  function(_super) {
    __extends(Request2, _super);
    function Request2(props) {
      var _this = this;
      var socket = {
        readable: false,
        destroyed: false,
        remoteAddress: props.remoteAddress,
        remotePort: Number(props.remotePort) || 0,
        resume: Function.prototype,
        destroy: Function.prototype,
        end: Function.prototype
      };
      _this = _super.call(this, socket) || this;
      if (props.captureLogs) {
        _this.logger = new Logger();
      }
      _this.__sk_event = props;
      _this.__sk__context = props.context;
      props.headers = props.headers || {};
      var url = "/";
      if (props.url) {
        try {
          var parsedUrl = new URL(props.url, "http://localhost");
          url = parsedUrl.pathname + parsedUrl.search;
        } catch (_a) {
          url = props.url.startsWith("/") ? props.url : "/" + props.url;
        }
      }
      Object.assign(_this, {
        url,
        complete: true,
        httpVersionMajor: "1",
        httpVersionMinor: "1",
        httpVersion: "1.1",
        httpMethod: props.method,
        method: props.method,
        headers: Object.keys(props.headers).reduce(function(obj, key) {
          var value = props.headers[key];
          if (Array.isArray(value)) {
            obj[key.toLowerCase()] = value.join(",");
          } else if (typeof value === "string" && value) {
            obj[key.toLowerCase()] = value;
          }
          return obj;
        }, {})
      });
      _this.rawHeaders = Object.keys(props.headers).reduce(function(array, key) {
        var value = props.headers[key];
        if (Array.isArray(value)) {
          value.forEach(function(v) {
            array.push(key, v);
          });
        } else if (value) {
          array.push(key, value);
        }
        return array;
      }, []);
      _this.pipe = function(destination) {
        var s = new Readable();
        s.push(props.body);
        s.push(null);
        s.pipe(destination);
        return destination;
      };
      registerEmitters(_this, props);
      return _this;
    }
    Object.defineProperty(Request2.prototype, "body", {
      get: function() {
        var _a;
        return ((_a = this.__sk_event) === null || _a === void 0 ? void 0 : _a.body) || "";
      },
      enumerable: false,
      configurable: true
    });
    return Request2;
  }(http.IncomingMessage)
);
var registerEmitters = function(obj, props) {
  var originalListener = obj.on;
  obj.on = function() {
    var args = [];
    for (var _i = 0; _i < arguments.length; _i++) {
      args[_i] = arguments[_i];
    }
    var event = args.shift();
    var listener = args.shift();
    if (event === "data") {
      listener(props.body);
    } else if (event === "end") {
      listener();
    } else if (event && listener) {
      originalListener.bind.apply(originalListener, __spreadArray([obj, event, listener], args, false));
    }
    return obj;
  };
};
var httpParser = {};
var hasRequiredHttpParser;
function requireHttpParser() {
  if (hasRequiredHttpParser) return httpParser;
  hasRequiredHttpParser = 1;
  httpParser.HTTPParser = HTTPParser;
  function HTTPParser(type) {
    if (type !== void 0 && type !== HTTPParser.REQUEST && type !== HTTPParser.RESPONSE) {
      throw new Error("type must be REQUEST or RESPONSE");
    }
    if (type === void 0) ;
    else {
      this.initialize(type);
    }
    this.maxHeaderSize = HTTPParser.maxHeaderSize;
  }
  HTTPParser.prototype.initialize = function(type, async_resource) {
    if (type !== HTTPParser.REQUEST && type !== HTTPParser.RESPONSE) {
      throw new Error("type must be REQUEST or RESPONSE");
    }
    this.type = type;
    this.state = type + "_LINE";
    this.info = {
      headers: [],
      upgrade: false
    };
    this.trailers = [];
    this.line = "";
    this.isChunked = false;
    this.connection = "";
    this.headerSize = 0;
    this.body_bytes = null;
    this.isUserCall = false;
    this.hadError = false;
  };
  HTTPParser.encoding = "ascii";
  HTTPParser.maxHeaderSize = 80 * 1024;
  HTTPParser.REQUEST = "REQUEST";
  HTTPParser.RESPONSE = "RESPONSE";
  var kOnHeaders = HTTPParser.kOnHeaders = 1;
  var kOnHeadersComplete = HTTPParser.kOnHeadersComplete = 2;
  var kOnBody = HTTPParser.kOnBody = 3;
  var kOnMessageComplete = HTTPParser.kOnMessageComplete = 4;
  HTTPParser.prototype[kOnHeaders] = HTTPParser.prototype[kOnHeadersComplete] = HTTPParser.prototype[kOnBody] = HTTPParser.prototype[kOnMessageComplete] = function() {
  };
  var compatMode0_12 = true;
  Object.defineProperty(HTTPParser, "kOnExecute", {
    get: function() {
      compatMode0_12 = false;
      return 99;
    }
  });
  var methods = httpParser.methods = HTTPParser.methods = [
    "DELETE",
    "GET",
    "HEAD",
    "POST",
    "PUT",
    "CONNECT",
    "OPTIONS",
    "TRACE",
    "COPY",
    "LOCK",
    "MKCOL",
    "MOVE",
    "PROPFIND",
    "PROPPATCH",
    "SEARCH",
    "UNLOCK",
    "BIND",
    "REBIND",
    "UNBIND",
    "ACL",
    "REPORT",
    "MKACTIVITY",
    "CHECKOUT",
    "MERGE",
    "M-SEARCH",
    "NOTIFY",
    "SUBSCRIBE",
    "UNSUBSCRIBE",
    "PATCH",
    "PURGE",
    "MKCALENDAR",
    "LINK",
    "UNLINK",
    "SOURCE"
  ];
  var method_connect = methods.indexOf("CONNECT");
  HTTPParser.prototype.reinitialize = HTTPParser;
  HTTPParser.prototype.close = HTTPParser.prototype.pause = HTTPParser.prototype.resume = HTTPParser.prototype.remove = HTTPParser.prototype.free = function() {
  };
  HTTPParser.prototype._compatMode0_11 = false;
  HTTPParser.prototype.getAsyncId = function() {
    return 0;
  };
  var headerState = {
    REQUEST_LINE: true,
    RESPONSE_LINE: true,
    HEADER: true
  };
  HTTPParser.prototype.execute = function(chunk, start, length) {
    if (!(this instanceof HTTPParser)) {
      throw new TypeError("not a HTTPParser");
    }
    start = start || 0;
    length = typeof length === "number" ? length : chunk.length;
    this.chunk = chunk;
    this.offset = start;
    var end = this.end = start + length;
    try {
      while (this.offset < end) {
        if (this[this.state]()) {
          break;
        }
      }
    } catch (err) {
      if (this.isUserCall) {
        throw err;
      }
      this.hadError = true;
      return err;
    }
    this.chunk = null;
    length = this.offset - start;
    if (headerState[this.state]) {
      this.headerSize += length;
      if (this.headerSize > (this.maxHeaderSize || HTTPParser.maxHeaderSize)) {
        return new Error("max header size exceeded");
      }
    }
    return length;
  };
  var stateFinishAllowed = {
    REQUEST_LINE: true,
    RESPONSE_LINE: true,
    BODY_RAW: true
  };
  HTTPParser.prototype.finish = function() {
    if (this.hadError) {
      return;
    }
    if (!stateFinishAllowed[this.state]) {
      return new Error("invalid state for EOF");
    }
    if (this.state === "BODY_RAW") {
      this.userCall()(this[kOnMessageComplete]());
    }
  };
  HTTPParser.prototype.consume = HTTPParser.prototype.unconsume = HTTPParser.prototype.getCurrentBuffer = function() {
  };
  HTTPParser.prototype.userCall = function() {
    this.isUserCall = true;
    var self = this;
    return function(ret) {
      self.isUserCall = false;
      return ret;
    };
  };
  HTTPParser.prototype.nextRequest = function() {
    this.userCall()(this[kOnMessageComplete]());
    this.reinitialize(this.type);
  };
  HTTPParser.prototype.consumeLine = function() {
    var end = this.end, chunk = this.chunk;
    for (var i = this.offset; i < end; i++) {
      if (chunk[i] === 10) {
        var line = this.line + chunk.toString(HTTPParser.encoding, this.offset, i);
        if (line.charAt(line.length - 1) === "\r") {
          line = line.substr(0, line.length - 1);
        }
        this.line = "";
        this.offset = i + 1;
        return line;
      }
    }
    this.line += chunk.toString(HTTPParser.encoding, this.offset, this.end);
    this.offset = this.end;
  };
  var headerExp = /^([^: \t]+):[ \t]*((?:.*[^ \t])|)/;
  var headerContinueExp = /^[ \t]+(.*[^ \t])/;
  HTTPParser.prototype.parseHeader = function(line, headers) {
    if (line.indexOf("\r") !== -1) {
      throw parseErrorCode("HPE_LF_EXPECTED");
    }
    var match2 = headerExp.exec(line);
    var k = match2 && match2[1];
    if (k) {
      headers.push(k);
      headers.push(match2[2]);
    } else {
      var matchContinue = headerContinueExp.exec(line);
      if (matchContinue && headers.length) {
        if (headers[headers.length - 1]) {
          headers[headers.length - 1] += " ";
        }
        headers[headers.length - 1] += matchContinue[1];
      }
    }
  };
  var requestExp = /^([A-Z-]+) ([^ ]+) HTTP\/(\d)\.(\d)$/;
  HTTPParser.prototype.REQUEST_LINE = function() {
    var line = this.consumeLine();
    if (!line) {
      return;
    }
    var match2 = requestExp.exec(line);
    if (match2 === null) {
      throw parseErrorCode("HPE_INVALID_CONSTANT");
    }
    this.info.method = this._compatMode0_11 ? match2[1] : methods.indexOf(match2[1]);
    if (this.info.method === -1) {
      throw new Error("invalid request method");
    }
    this.info.url = match2[2];
    this.info.versionMajor = +match2[3];
    this.info.versionMinor = +match2[4];
    this.body_bytes = 0;
    this.state = "HEADER";
  };
  var responseExp = /^HTTP\/(\d)\.(\d) (\d{3}) ?(.*)$/;
  HTTPParser.prototype.RESPONSE_LINE = function() {
    var line = this.consumeLine();
    if (!line) {
      return;
    }
    var match2 = responseExp.exec(line);
    if (match2 === null) {
      throw parseErrorCode("HPE_INVALID_CONSTANT");
    }
    this.info.versionMajor = +match2[1];
    this.info.versionMinor = +match2[2];
    var statusCode = this.info.statusCode = +match2[3];
    this.info.statusMessage = match2[4];
    if ((statusCode / 100 | 0) === 1 || statusCode === 204 || statusCode === 304) {
      this.body_bytes = 0;
    }
    this.state = "HEADER";
  };
  HTTPParser.prototype.shouldKeepAlive = function() {
    if (this.info.versionMajor > 0 && this.info.versionMinor > 0) {
      if (this.connection.indexOf("close") !== -1) {
        return false;
      }
    } else if (this.connection.indexOf("keep-alive") === -1) {
      return false;
    }
    if (this.body_bytes !== null || this.isChunked) {
      return true;
    }
    return false;
  };
  HTTPParser.prototype.HEADER = function() {
    var line = this.consumeLine();
    if (line === void 0) {
      return;
    }
    var info = this.info;
    if (line) {
      this.parseHeader(line, info.headers);
    } else {
      var headers = info.headers;
      var hasContentLength = false;
      var currentContentLengthValue;
      var hasUpgradeHeader = false;
      for (var i = 0; i < headers.length; i += 2) {
        switch (headers[i].toLowerCase()) {
          case "transfer-encoding":
            this.isChunked = headers[i + 1].toLowerCase() === "chunked";
            break;
          case "content-length":
            currentContentLengthValue = +headers[i + 1];
            if (hasContentLength) {
              if (currentContentLengthValue !== this.body_bytes) {
                throw parseErrorCode("HPE_UNEXPECTED_CONTENT_LENGTH");
              }
            } else {
              hasContentLength = true;
              this.body_bytes = currentContentLengthValue;
            }
            break;
          case "connection":
            this.connection += headers[i + 1].toLowerCase();
            break;
          case "upgrade":
            hasUpgradeHeader = true;
            break;
        }
      }
      if (this.isChunked && hasContentLength) {
        hasContentLength = false;
        this.body_bytes = null;
      }
      if (hasUpgradeHeader && this.connection.indexOf("upgrade") != -1) {
        info.upgrade = this.type === HTTPParser.REQUEST || info.statusCode === 101;
      } else {
        info.upgrade = info.method === method_connect;
      }
      if (this.isChunked && info.upgrade) {
        this.isChunked = false;
      }
      info.shouldKeepAlive = this.shouldKeepAlive();
      var skipBody;
      if (compatMode0_12) {
        skipBody = this.userCall()(this[kOnHeadersComplete](info));
      } else {
        skipBody = this.userCall()(this[kOnHeadersComplete](
          info.versionMajor,
          info.versionMinor,
          info.headers,
          info.method,
          info.url,
          info.statusCode,
          info.statusMessage,
          info.upgrade,
          info.shouldKeepAlive
        ));
      }
      if (skipBody === 2) {
        this.nextRequest();
        return true;
      } else if (this.isChunked && !skipBody) {
        this.state = "BODY_CHUNKHEAD";
      } else if (skipBody || this.body_bytes === 0) {
        this.nextRequest();
        return info.upgrade;
      } else if (this.body_bytes === null) {
        this.state = "BODY_RAW";
      } else {
        this.state = "BODY_SIZED";
      }
    }
  };
  HTTPParser.prototype.BODY_CHUNKHEAD = function() {
    var line = this.consumeLine();
    if (line === void 0) {
      return;
    }
    this.body_bytes = parseInt(line, 16);
    if (!this.body_bytes) {
      this.state = "BODY_CHUNKTRAILERS";
    } else {
      this.state = "BODY_CHUNK";
    }
  };
  HTTPParser.prototype.BODY_CHUNK = function() {
    var length = Math.min(this.end - this.offset, this.body_bytes);
    this.userCall()(this[kOnBody](this.chunk.slice(this.offset, this.offset + length), 0, length));
    this.offset += length;
    this.body_bytes -= length;
    if (!this.body_bytes) {
      this.state = "BODY_CHUNKEMPTYLINE";
    }
  };
  HTTPParser.prototype.BODY_CHUNKEMPTYLINE = function() {
    var line = this.consumeLine();
    if (line === void 0) {
      return;
    }
    if (line !== "") {
      throw new Error("Expected empty line");
    }
    this.state = "BODY_CHUNKHEAD";
  };
  HTTPParser.prototype.BODY_CHUNKTRAILERS = function() {
    var line = this.consumeLine();
    if (line === void 0) {
      return;
    }
    if (line) {
      this.parseHeader(line, this.trailers);
    } else {
      if (this.trailers.length) {
        this.userCall()(this[kOnHeaders](this.trailers, ""));
      }
      this.nextRequest();
    }
  };
  HTTPParser.prototype.BODY_RAW = function() {
    this.userCall()(this[kOnBody](this.chunk.slice(this.offset, this.end), 0, this.end - this.offset));
    this.offset = this.end;
  };
  HTTPParser.prototype.BODY_SIZED = function() {
    var length = Math.min(this.end - this.offset, this.body_bytes);
    this.userCall()(this[kOnBody](this.chunk.slice(this.offset, this.offset + length), 0, length));
    this.offset += length;
    this.body_bytes -= length;
    if (!this.body_bytes) {
      this.nextRequest();
    }
  };
  ["Headers", "HeadersComplete", "Body", "MessageComplete"].forEach(function(name) {
    var k = HTTPParser["kOn" + name];
    Object.defineProperty(HTTPParser.prototype, "on" + name, {
      get: function() {
        return this[k];
      },
      set: function(to) {
        this._compatMode0_11 = true;
        method_connect = "CONNECT";
        return this[k] = to;
      }
    });
  });
  function parseErrorCode(code) {
    var err = new Error("Parse Error");
    err.code = code;
    return err;
  }
  return httpParser;
}
var httpParserExports = requireHttpParser();
var httpParse = function(buffer) {
  if (!buffer) {
    return { headers: {} };
  }
  var parser = new httpParserExports.HTTPParser(httpParserExports.HTTPParser.RESPONSE);
  var parsed = {
    headers: {}
  };
  parser.onHeadersComplete = function(info) {
    parsed.statusCode = info.statusCode;
    parsed.statusMessage = info.statusMessage;
    while (info.headers.length > 0) {
      var key = info.headers.shift();
      var val = info.headers.shift();
      if (key && val) {
        var lowerKey = key.toLowerCase();
        if (Array.isArray(parsed.headers[lowerKey])) {
          parsed.headers[lowerKey].push(val);
        } else if (parsed.headers[lowerKey]) {
          parsed.headers[lowerKey] = [parsed.headers[lowerKey], val];
        } else {
          parsed.headers[lowerKey] = val;
        }
      }
    }
  };
  var body = [];
  parser.onBody = function(chunk, offset, length) {
    body.push(chunk.slice(offset, offset + length));
  };
  parser.execute(buffer);
  parser.finish();
  parser.close();
  parsed.buffer = Buffer.concat(body);
  return parsed;
};
var ResponseStream = (
  /** @class */
  function(_super) {
    __extends(ResponseStream2, _super);
    function ResponseStream2() {
      var _this = _super !== null && _super.apply(this, arguments) || this;
      _this.buffer = [];
      return _this;
    }
    ResponseStream2.prototype._write = function(chunk, encoding, next) {
      this.buffer.push(Buffer.from(chunk, encoding));
      next();
    };
    ResponseStream2.prototype._writev = function(chunks, next) {
      var _this = this;
      chunks.forEach(function(d) {
        _this.buffer.push(Buffer.from(d.chunk, d.encoding));
      });
      next();
    };
    ResponseStream2.prototype._destroy = function(err, callback) {
      this.buffer = [];
      callback(err);
    };
    return ResponseStream2;
  }(stream$1.Writable)
);
var createStream = function(req) {
  var _a, _b;
  var stream2 = new ResponseStream({ highWaterMark: Number.MAX_VALUE });
  Object.defineProperties(stream2, {
    remoteFamily: { value: "IPv4" },
    remotePort: { value: req.socket.remotePort || ((_a = req.connection) === null || _a === void 0 ? void 0 : _a.remotePort) },
    remoteAddress: {
      value: req.socket.remoteAddress || ((_b = req.connection) === null || _b === void 0 ? void 0 : _b.remoteAddress)
    }
  });
  return stream2;
};
var Response = (
  /** @class */
  function(_super) {
    __extends(Response2, _super);
    function Response2(req) {
      var _this = _super.call(this, req) || this;
      _this.shouldKeepAlive = false;
      _this.chunkedEncoding = false;
      _this.useChunkedEncodingByDefault = false;
      var stream2 = createStream(req);
      _this.assignSocket(stream2);
      var response = _this;
      _this.on("prefinish", function() {
        var _a, _b, _c;
        var parsed = httpParse(Buffer.concat(stream2.buffer));
        parsed.headers.connection = "close";
        ["accept-ranges", "transfer-encoding"].forEach(function(ignoredKey) {
          delete parsed.headers[ignoredKey];
        });
        (_a = response.socket) === null || _a === void 0 ? void 0 : _a.destroy();
        response.emit("sk-end", {
          buffer: (_b = parsed.buffer) === null || _b === void 0 ? void 0 : _b.toString("base64"),
          headers: parsed.headers,
          statusMessage: parsed.statusMessage || "OK",
          status: parsed.statusCode || 200,
          logs: (_c = req.logger) === null || _c === void 0 ? void 0 : _c.logs()
        });
      });
      return _this;
    }
    return Response2;
  }(ServerResponse)
);
const pathToRegExp = (path2) => {
  const pattern = path2.replace(/\./g, "\\.").replace(/\//g, "/").replace(/\?/g, "\\?").replace(/\/+$/, "").replace(/\*+/g, ".*").replace(/:([^\d|^\/][a-zA-Z0-9_]*(?=(?:\/|\\.)|$))/g, (_, paramName) => `(?<${paramName}>[^/]+?)`).concat("(\\/|$)");
  return new RegExp(pattern, "gi");
};
const match = (path2, url) => {
  const expression = path2 instanceof RegExp ? path2 : pathToRegExp(path2);
  const match2 = expression.exec(url) || false;
  const matches = path2 instanceof RegExp ? !!match2 : !!match2 && match2[0] === match2.input;
  return {
    matches,
    params: match2 && matches ? match2.groups || null : null
  };
};
var walkTree = function(directory, tree) {
  if (tree === void 0) {
    tree = [];
  }
  var results = [];
  var files = fs.readdirSync(directory);
  var dirs = [];
  for (var _i = 0, files_1 = files; _i < files_1.length; _i++) {
    var fileName = files_1[_i];
    var filePath = path.join(directory, fileName);
    var fileStats = fs.statSync(filePath);
    if (!fileStats.isDirectory()) {
      results.push({
        name: fileName,
        path: directory,
        rel: path.join.apply(path, __spreadArray(__spreadArray([], tree, false), [fileName], false))
      });
    } else {
      dirs.push(fileName);
    }
  }
  for (var _a = 0, dirs_1 = dirs; _a < dirs_1.length; _a++) {
    var dirName = dirs_1[_a];
    var filePath = path.join(directory, dirName);
    results.push.apply(results, walkTree(filePath, __spreadArray(__spreadArray([], tree, true), [dirName], false)));
  }
  return results;
};
var parseFileName = function(fileName) {
  var pieces = fileName.split(".");
  if (pieces.length <= 2) {
    return { name: pieces[0], method: "all" };
  }
  return {
    name: pieces[0],
    method: pieces[pieces.length - 2].toLowerCase()
  };
};
var fileToRoute = function(file) {
  var _a;
  var fileName = (_a = file.split(path.sep).pop()) === null || _a === void 0 ? void 0 : _a.split(".")[0];
  var normalized = file.replace(/\[([a-zA-Z0-9_\.:-]*)\]/g, ":$1");
  if (fileName === "index") {
    normalized = path.dirname(normalized);
  } else {
    normalized = normalized.split(".")[0];
  }
  return path.join(path.sep, normalized);
};
var matchPath = function(files, requestPath, requestMethod) {
  if (requestMethod === void 0) {
    requestMethod = "get";
  }
  var method = requestMethod.toLowerCase();
  for (var _i = 0, files_2 = files; _i < files_2.length; _i++) {
    var file = files_2[_i];
    var parsed = parseFileName(file.name);
    if (file.name.startsWith("_") || file.rel.indexOf("".concat(path.sep, "_")) > -1) {
      continue;
    }
    if (file.name.includes(".spec.")) {
      continue;
    }
    if (parsed.method !== "all" && parsed.method !== method) {
      continue;
    }
    var route = fileToRoute(file.rel);
    if (match(route, requestPath).matches) {
      return file;
    }
  }
};
var cachedFiles;
var invokeApiHandler = function(handler2, req, res) {
  var _a, _b;
  var fn = [
    handler2 === null || handler2 === void 0 ? void 0 : handler2.handler,
    (_a = handler2 === null || handler2 === void 0 ? void 0 : handler2.default) === null || _a === void 0 ? void 0 : _a.handler,
    handler2 === null || handler2 === void 0 ? void 0 : handler2.default,
    (_b = handler2 === null || handler2 === void 0 ? void 0 : handler2.default) === null || _b === void 0 ? void 0 : _b.default,
    // This is a hack for commonjsjs modules that export a default fn as a property of the default export.
    handler2
  ].find(function(f) {
    return typeof f === "function";
  });
  return Promise.resolve(fn(req, res)).then(function(r) {
    if (typeof r !== "undefined" && typeof r === "object") {
      var isBodyAnObject = typeof r.body === "object";
      var headers = {};
      if (isBodyAnObject) {
        headers["Content-Type"] = "application/json";
      }
      return {
        body: typeof r.body === "string" ? r.body : JSON.stringify(r.body),
        headers: __assign(__assign({}, headers), r.headers),
        status: r.statusCode || r.status
      };
    }
  });
};
var handleApi = function(event, apiDir) {
  if (typeof cachedFiles === "undefined") {
    cachedFiles = walkTree(apiDir);
  }
  return new Promise(function(resolve) {
    return __awaiter(void 0, void 0, void 0, function() {
      var req, res, requestPath, apiPrefix, file, mod, ret, e_1;
      var _a, _b, _c;
      return __generator(this, function(_d) {
        switch (_d.label) {
          case 0:
            req = new Request(event);
            res = new Response(req);
            res.on("sk-end", function(data) {
              resolve(data);
            });
            requestPath = "/" + ((req.url || "").split(/[\?#]/)[0] || "").replace(/^\/+/, "");
            apiPrefix = ((_a = req.__sk__context) === null || _a === void 0 ? void 0 : _a.apiPrefix) || "";
            if (apiPrefix && requestPath.startsWith(apiPrefix)) {
              requestPath = requestPath.slice(apiPrefix.length) || "/";
            }
            file = matchPath(cachedFiles, requestPath, req.method);
            if (!file) return [3, 5];
            _d.label = 1;
          case 1:
            _d.trys.push([1, 4, , 5]);
            return [4, import(path.join(file.path, file.name))];
          case 2:
            mod = _d.sent();
            return [4, invokeApiHandler(mod, req, res)];
          case 3:
            ret = _d.sent();
            if (ret) {
              resolve(__assign(__assign({}, ret), { logs: (_b = req.logger) === null || _b === void 0 ? void 0 : _b.logs() }));
            }
            return [
              2
              /*return*/
            ];
          case 4:
            e_1 = _d.sent();
            if (e_1 instanceof Error && ((_c = e_1.message) === null || _c === void 0 ? void 0 : _c.includes("handler is not a function"))) {
              console.error("API Function does not export a default method. See https://www.stormkit.io/docs/features/writing-api for more information.");
            } else {
              console.error(e_1);
            }
            return [3, 5];
          case 5:
            res.writeHead(404, "Not found");
            res.end();
            return [
              2
              /*return*/
            ];
        }
      });
    });
  });
};
var handleError = function(callback) {
  return function(e) {
    var msg = e && e.message ? e.message : void 0;
    var stack = e && e.stack ? e.stack : void 0;
    if (e && !msg && typeof e === "string") {
      msg = e;
    }
    if (typeof msg !== "string") {
      msg = JSON.stringify(e);
    }
    return callback(null, {
      status: 500,
      errorMessage: msg,
      errorStack: stack
    });
  };
};
var serverlessApi = function(dirname) {
  return function(event, context, callback) {
    return __awaiter(void 0, void 0, void 0, function() {
      var e, _a, _b, e_1;
      return __generator(this, function(_c) {
        switch (_c.label) {
          case 0:
            context.callbackWaitsForEmptyEventLoop = false;
            e = Buffer.isBuffer(event) ? JSON.parse(event.toString()) : event;
            _c.label = 1;
          case 1:
            _c.trys.push([1, 3, , 4]);
            _a = callback;
            _b = [null];
            return [4, handleApi(e, dirname)];
          case 2:
            _a.apply(void 0, _b.concat([_c.sent()]));
            return [3, 4];
          case 3:
            e_1 = _c.sent();
            handleError(callback)(e_1);
            return [3, 4];
          case 4:
            return [
              2
              /*return*/
            ];
        }
      });
    });
  };
};
const handler = serverlessApi(import.meta.dirname);
export {
  handler
};
