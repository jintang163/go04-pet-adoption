const TOKEN_KEY = "pa_token";

function token() { return localStorage.getItem(TOKEN_KEY) || ""; }
function setToken(t) { localStorage.setItem(TOKEN_KEY, t); }
function clearToken() { localStorage.removeItem(TOKEN_KEY); }

async function api(path, opts = {}) {
  const headers = Object.assign({ "Content-Type": "application/json" }, opts.headers || {});
  const t = token();
  if (t) headers.Authorization = "Bearer " + t;
  const res = await fetch(path, Object.assign({}, opts, { headers }));
  if (res.status === 204) return null;
  const text = await res.text();
  let data = null;
  try { data = text ? JSON.parse(text) : null; } catch { data = { message: text }; }
  if (!res.ok) {
    const msg = (data && (data.message || data.code)) || ("HTTP " + res.status);
    throw new Error(msg);
  }
  return data;
}

function requireLogin() {
  if (!token()) { location.href = "/login"; return false; }
  return true;
}

async function currentMe() {
  return api("/api/auth/me");
}

function logout() {
  api("/api/auth/logout", { method: "POST" }).catch(() => {});
  clearToken();
  location.href = "/login";
}

function $(id) { return document.getElementById(id); }

function fillUserBar(me) {
  const el = $("userbar");
  if (!el || !me) return;
  const u = me.user || me;
  el.innerHTML = `<span>${u.display_name || u.username} · 信用 ${u.credit_score}（${u.credit_level}）</span>
    <a href="/app">待领养</a> <a href="/me">我的</a>
    ${u.role === "staff" || u.role === "admin" ? '<a href="/staff">救助站</a>' : ""}
    ${u.role === "admin" ? '<a href="/admin">后台</a>' : ""}
    <button class="btn btn-ghost" onclick="logout()">退出</button>`;
}

function speciesLabel(id) {
  const m = {dog:"狗",cat:"猫",rabbit:"兔",bird:"鸟",other:"其他"};
  return m[id] || id;
}
function sizeLabel(id) {
  const m = {small:"小型",medium:"中型",large:"大型"};
  return m[id] || id;
}
function statusLabel(id) {
  const m = {draft:"草稿",published:"待领养",reserved:"已预留",adopted:"已领养",unavailable:"暂不可领养",deceased:"已死亡",
    pending:"待审",under_review:"审核中",waitlisted:"候补",approved:"已录取",rejected:"已拒绝",withdrawn:"已撤回",completed:"已交接",revoked:"已撤销",
    scheduled:"待上门",missed:"缺访",cancelled:"已取消"};
  return m[id] || id;
}

window.PA = { api, token, setToken, clearToken, requireLogin, currentMe, logout, $, fillUserBar, speciesLabel, sizeLabel, statusLabel };
