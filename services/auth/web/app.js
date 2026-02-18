const API = "http://localhost:8081";

function log(id, msg, cls = "log") {
  const el = document.getElementById(id);
  if (cls === "error" || cls === "success") el.innerHTML = "";
  const p = document.createElement("pre");
  p.className = cls;
  p.textContent = typeof msg === "string" ? msg : JSON.stringify(msg, null, 2);
  el.appendChild(p);
}

// base64url helpers
function bufToBase64url(buf) {
  const bytes = new Uint8Array(buf);
  let str = "";
  for (const b of bytes) str += String.fromCharCode(b);
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

function base64urlToBuf(b64) {
  const str = b64.replace(/-/g, "+").replace(/_/g, "/");
  const pad = str.length % 4 === 0 ? "" : "=".repeat(4 - (str.length % 4));
  const binary = atob(str + pad);
  const buf = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) buf[i] = binary.charCodeAt(i);
  return buf.buffer;
}

async function registerBegin() {
  const email = document.getElementById("reg-email").value;
  try {
    const res = await fetch(`${API}/api/register/begin`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ email }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    log("reg-log", "Check server console for verification code.", "success");
    log("reg-log", data);
  } catch (e) {
    log("reg-log", e.message, "error");
  }
}

async function registerVerify() {
  const email = document.getElementById("reg-email").value;
  const code = document.getElementById("reg-code").value;
  try {
    const res = await fetch(`${API}/api/register/verify`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ email, code }),
    });
    if (!res.ok) throw new Error(await res.text());
    const options = await res.json();
    log("reg-log", "Got credential creation options:");
    log("reg-log", options);

    // Decode for WebAuthn API
    options.publicKey.challenge = base64urlToBuf(options.publicKey.challenge);
    options.publicKey.user.id = base64urlToBuf(options.publicKey.user.id);
    if (options.publicKey.excludeCredentials) {
      options.publicKey.excludeCredentials =
        options.publicKey.excludeCredentials.map((c) => ({
          ...c,
          id: base64urlToBuf(c.id),
        }));
    }

    log("reg-log", "Creating passkey...", "log");
    const credential = await navigator.credentials.create(options);

    const body = {
      id: credential.id,
      rawId: bufToBase64url(credential.rawId),
      type: credential.type,
      response: {
        attestationObject: bufToBase64url(
          credential.response.attestationObject,
        ),
        clientDataJSON: bufToBase64url(credential.response.clientDataJSON),
      },
    };

    if (credential.response.getTransports) {
      body.response.transports = credential.response.getTransports();
    }

    const finishRes = await fetch(`${API}/api/register/finish`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(body),
    });
    if (!finishRes.ok) throw new Error(await finishRes.text());
    const finishData = await finishRes.json();
    log("reg-log", "Registration complete!", "success");
    log("reg-log", finishData);
  } catch (e) {
    log("reg-log", e.message, "error");
  }
}

async function loginBegin() {
  try {
    const res = await fetch(`${API}/api/login/begin`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
    });
    if (!res.ok) throw new Error(await res.text());
    const options = await res.json();
    log("login-log", "Got assertion options:");
    log("login-log", options);

    // Decode for WebAuthn API
    options.publicKey.challenge = base64urlToBuf(options.publicKey.challenge);
    if (options.publicKey.allowCredentials) {
      options.publicKey.allowCredentials =
        options.publicKey.allowCredentials.map((c) => ({
          ...c,
          id: base64urlToBuf(c.id),
        }));
    }

    log("login-log", "Requesting passkey...", "log");
    const assertion = await navigator.credentials.get(options);

    const body = {
      id: assertion.id,
      rawId: bufToBase64url(assertion.rawId),
      type: assertion.type,
      response: {
        authenticatorData: bufToBase64url(assertion.response.authenticatorData),
        clientDataJSON: bufToBase64url(assertion.response.clientDataJSON),
        signature: bufToBase64url(assertion.response.signature),
        userHandle: bufToBase64url(assertion.response.userHandle),
      },
    };

    const finishRes = await fetch(`${API}/api/login/finish`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify(body),
    });
    if (!finishRes.ok) throw new Error(await finishRes.text());
    const finishData = await finishRes.json();
    log("login-log", "Login successful!", "success");
    log("login-log", finishData);
  } catch (e) {
    log("login-log", e.message, "error");
  }
}
