// ===== State =====
let currentPage = 'dashboard';

// ===== Helpers =====
function formatBytes(bytes) {
  if (!bytes || bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i];
}

function showLogin() {
  document.getElementById('login-page').classList.remove('hidden');
  document.getElementById('app').classList.add('hidden');
}

function showApp() {
  document.getElementById('login-page').classList.add('hidden');
  document.getElementById('app').classList.remove('hidden');
  loadPage('dashboard');
}

// ===== Login =====
document.getElementById('login-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const user = document.getElementById('login-user').value;
  const pass = document.getElementById('login-pass').value;
  const errEl = document.getElementById('login-error');
  errEl.textContent = '';

  const res = await API.login(user, pass);
  if (res && res.success) {
    API.token = res.data.token;
    localStorage.setItem('o-ui-token', res.data.token);
    document.getElementById('username-display').textContent = res.data.username;
    showApp();
  } else {
    errEl.textContent = res ? res.message : 'Connection failed';
  }
});

// ===== Logout =====
document.getElementById('logout-btn').addEventListener('click', (e) => {
  e.preventDefault();
  API.token = '';
  localStorage.removeItem('o-ui-token');
  showLogin();
});

// ===== Navigation =====
document.querySelectorAll('.nav-item').forEach(item => {
  item.addEventListener('click', (e) => {
    e.preventDefault();
    const page = item.dataset.page;
    loadPage(page);
  });
});

function loadPage(page) {
  currentPage = page;
  document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));
  document.querySelector(`.nav-item[data-page="${page}"]`)?.classList.add('active');
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  document.getElementById('page-' + page)?.classList.add('active');

  const titles = { dashboard: 'Dashboard', inbounds: 'Inbound Management', outbounds: 'Outbound Management', nodes: 'Node Management', settings: 'Settings' };
  document.getElementById('page-title').textContent = titles[page] || page;

  switch (page) {
    case 'dashboard': loadDashboard(); break;
    case 'inbounds': loadInbounds(); break;
    case 'outbounds': loadOutbounds(); break;
    case 'nodes': loadNodes(); break;
    case 'settings': loadSettings(); break;
  }
}

// ===== Dashboard =====
async function loadDashboard() {
  const [stats, system, core] = await Promise.all([
    API.getStats(), API.getSystemInfo(), API.getCoreStatus()
  ]);

  if (stats?.success) {
    document.getElementById('stat-total-up').textContent = formatBytes(stats.data.total_up);
    document.getElementById('stat-total-down').textContent = formatBytes(stats.data.total_down);
  }
  if (core?.success) {
    document.getElementById('stat-inbounds').textContent = core.data.inbounds;
    document.getElementById('stat-nodes').textContent = core.data.outbounds;
    updateCoreStatus(core.data.running);
  }
  if (system?.success) {
    const d = system.data;
    document.getElementById('system-info-table').innerHTML = `
      <tr><td>Hostname</td><td>${d.hostname}</td></tr>
      <tr><td>OS / Arch</td><td>${d.os} / ${d.arch}</td></tr>
      <tr><td>Go Version</td><td>${d.go_version}</td></tr>
      <tr><td>CPU Cores</td><td>${d.num_cpu}</td></tr>
      <tr><td>Goroutines</td><td>${d.num_goroutine}</td></tr>
      <tr><td>Memory Alloc</td><td>${formatBytes(d.mem_alloc)}</td></tr>
      <tr><td>Memory Sys</td><td>${formatBytes(d.mem_sys)}</td></tr>
      <tr><td>Uptime</td><td>${d.uptime}</td></tr>
    `;
  }
}

function updateCoreStatus(running) {
  const badge = document.getElementById('core-status-badge');
  const btn = document.getElementById('core-toggle-btn');
  if (running) {
    badge.className = 'badge badge-success';
    badge.textContent = 'Running';
    btn.textContent = 'Stop Core';
    btn.className = 'btn btn-sm btn-danger';
  } else {
    badge.className = 'badge badge-danger';
    badge.textContent = 'Stopped';
    btn.textContent = 'Start Core';
    btn.className = 'btn btn-sm btn-primary';
  }
}

document.getElementById('core-toggle-btn').addEventListener('click', async () => {
  const status = await API.getCoreStatus();
  if (status?.success && status.data.running) {
    await API.stopCore();
  } else {
    await API.startCore();
  }
  loadDashboard();
});

// ===== Inbounds =====
async function loadInbounds() {
  const res = await API.getInbounds();
  const tbody = document.getElementById('inbounds-table');
  if (!res?.success || !res.data) { tbody.innerHTML = '<tr><td colspan="7" style="text-align:center;color:var(--text-secondary)">No inbounds</td></tr>'; return; }
  tbody.innerHTML = res.data.map(inb => `
    <tr>
      <td><strong>${inb.tag}</strong></td>
      <td>${inb.protocol}</td>
      <td>${inb.port}</td>
      <td>${inb.listen}</td>
      <td><span class="badge ${inb.enabled ? 'badge-success' : 'badge-danger'}">${inb.enabled ? 'Active' : 'Disabled'}</span></td>
      <td>↑${formatBytes(inb.traffic_up)} ↓${formatBytes(inb.traffic_down)}</td>
      <td><button class="btn btn-sm btn-danger" onclick="deleteInbound(${inb.id})">Delete</button></td>
    </tr>
  `).join('');
}

async function deleteInbound(id) {
  if (!confirm('Delete this inbound?')) return;
  await API.deleteInbound(id);
  loadInbounds();
}

function showInboundModal() {
  showModal('Add Inbound', `
    <div class="input-group"><label>Tag</label><input type="text" id="m-tag" placeholder="inbound-01"></div>
    <div class="form-row">
      <div class="input-group"><label>Protocol</label>
        <select id="m-protocol"><option value="vmess">VMess</option><option value="vless">VLESS</option><option value="trojan">Trojan</option><option value="shadowsocks">Shadowsocks</option><option value="socks">SOCKS</option><option value="http">HTTP</option><option value="dccp">DCCP</option></select>
      </div>
      <div class="input-group"><label>Port</label><input type="number" id="m-port" value="443"></div>
    </div>
    <div class="input-group"><label>Listen</label><input type="text" id="m-listen" value="0.0.0.0"></div>
    <div class="input-group"><label>Remark</label><input type="text" id="m-remark" placeholder="My server"></div>
  `, async () => {
    const data = {
      tag: document.getElementById('m-tag').value,
      protocol: document.getElementById('m-protocol').value,
      port: parseInt(document.getElementById('m-port').value),
      listen: document.getElementById('m-listen').value,
      remark: document.getElementById('m-remark').value,
      enabled: true,
      settings: '{}',
      stream_settings: '{}'
    };
    const res = await API.addInbound(data);
    if (res?.success) { closeModal(); loadInbounds(); }
    else alert(res?.message || 'Failed');
  });
}

// ===== Outbounds =====
async function loadOutbounds() {
  const res = await API.getOutbounds();
  const tbody = document.getElementById('outbounds-table');
  if (!res?.success || !res.data) { tbody.innerHTML = '<tr><td colspan="5" style="text-align:center;color:var(--text-secondary)">No outbounds</td></tr>'; return; }
  tbody.innerHTML = res.data.map(o => `
    <tr>
      <td><strong>${o.tag}</strong></td>
      <td>${o.protocol}</td>
      <td><span class="badge ${o.enabled ? 'badge-success' : 'badge-danger'}">${o.enabled ? 'Active' : 'Disabled'}</span></td>
      <td>${o.remark || '-'}</td>
      <td><button class="btn btn-sm btn-danger" onclick="deleteOutbound(${o.id})">Delete</button></td>
    </tr>
  `).join('');
}

async function deleteOutbound(id) {
  if (!confirm('Delete this outbound?')) return;
  await API.deleteOutbound(id);
  loadOutbounds();
}

function showOutboundModal() {
  showModal('Add Outbound', `
    <div class="input-group"><label>Tag</label><input type="text" id="m-tag" placeholder="outbound-01"></div>
    <div class="form-row">
      <div class="input-group"><label>Protocol</label>
        <select id="m-protocol"><option value="freedom">Freedom</option><option value="blackhole">Blackhole</option><option value="vmess">VMess</option><option value="vless">VLESS</option><option value="trojan">Trojan</option><option value="shadowsocks">Shadowsocks</option><option value="dccp">DCCP</option></select>
      </div>
      <div class="input-group"><label>Remark</label><input type="text" id="m-remark"></div>
    </div>
  `, async () => {
    const data = {
      tag: document.getElementById('m-tag').value,
      protocol: document.getElementById('m-protocol').value,
      remark: document.getElementById('m-remark').value,
      enabled: true,
      settings: '{}',
      stream_settings: '{}'
    };
    const res = await API.addOutbound(data);
    if (res?.success) { closeModal(); loadOutbounds(); }
    else alert(res?.message || 'Failed');
  });
}

// ===== Nodes =====
async function loadNodes() {
  const res = await API.getNodes();
  const tbody = document.getElementById('nodes-table');
  if (!res?.success || !res.data) { tbody.innerHTML = '<tr><td colspan="8" style="text-align:center;color:var(--text-secondary)">No nodes</td></tr>'; return; }
  tbody.innerHTML = res.data.map(n => `
    <tr>
      <td><strong>${n.name}</strong></td>
      <td>${n.address}</td>
      <td>${n.port}</td>
      <td>${n.protocol}</td>
      <td>${n.group_name}</td>
      <td>${n.latency > 0 ? n.latency + 'ms' : n.latency === -1 ? '<span class="badge badge-danger">Timeout</span>' : '-'}</td>
      <td>↑${formatBytes(n.traffic_up)} ↓${formatBytes(n.traffic_down)}</td>
      <td>
        <button class="btn btn-sm btn-secondary" onclick="checkNode(${n.id})">Check</button>
        <button class="btn btn-sm btn-danger" onclick="deleteNode(${n.id})">Del</button>
      </td>
    </tr>
  `).join('');
}

async function deleteNode(id) {
  if (!confirm('Delete this node?')) return;
  await API.deleteNode(id);
  loadNodes();
}

async function checkNode(id) {
  const res = await API.checkNode(id);
  if (res?.success) alert('Latency: ' + res.data.latency + 'ms');
  else alert(res?.message || 'Check failed');
  loadNodes();
}

function showNodeModal() {
  showModal('Add Node', `
    <div class="input-group"><label>Name</label><input type="text" id="m-name" placeholder="Tokyo Server"></div>
    <div class="form-row">
      <div class="input-group"><label>Address</label><input type="text" id="m-addr" placeholder="1.2.3.4"></div>
      <div class="input-group"><label>Port</label><input type="number" id="m-port" value="443"></div>
    </div>
    <div class="form-row">
      <div class="input-group"><label>Protocol</label>
        <select id="m-protocol"><option value="vmess">VMess</option><option value="vless">VLESS</option><option value="trojan">Trojan</option><option value="shadowsocks">Shadowsocks</option></select>
      </div>
      <div class="input-group"><label>Group</label><input type="text" id="m-group" value="default"></div>
    </div>
    <div class="input-group"><label>UUID / Password</label><input type="text" id="m-uuid"></div>
  `, async () => {
    const data = {
      name: document.getElementById('m-name').value,
      address: document.getElementById('m-addr').value,
      port: parseInt(document.getElementById('m-port').value),
      protocol: document.getElementById('m-protocol').value,
      group_name: document.getElementById('m-group').value,
      uuid: document.getElementById('m-uuid').value,
      password: document.getElementById('m-uuid').value,
      enabled: true,
      settings: '{}'
    };
    const res = await API.addNode(data);
    if (res?.success) { closeModal(); loadNodes(); }
    else alert(res?.message || 'Failed');
  });
}

// ===== Settings =====
async function loadSettings() {
  const res = await API.getSettings();
  if (res?.success) {
    for (const [k, v] of Object.entries(res.data)) {
      const el = document.getElementById('set-' + k);
      if (el) el.value = v;
    }
  }
}

document.getElementById('settings-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const data = {};
  document.querySelectorAll('#settings-form input, #settings-form select').forEach(el => {
    const key = el.id.replace('set-', '');
    data[key] = el.value;
  });
  const res = await API.updateSettings(data);
  alert(res?.success ? 'Settings saved!' : res?.message || 'Failed');
});

document.getElementById('password-form').addEventListener('submit', async (e) => {
  e.preventDefault();
  const oldP = document.getElementById('old-pass').value;
  const newP = document.getElementById('new-pass').value;
  const res = await API.changePassword(oldP, newP);
  if (res?.success) { alert('Password changed!'); document.getElementById('old-pass').value = ''; document.getElementById('new-pass').value = ''; }
  else alert(res?.message || 'Failed');
});

// ===== Modal =====
function showModal(title, bodyHTML, onSave) {
  document.getElementById('modal-title').textContent = title;
  document.getElementById('modal-body').innerHTML = bodyHTML;
  document.getElementById('modal-overlay').classList.remove('hidden');
  const saveBtn = document.getElementById('modal-save');
  saveBtn.onclick = onSave;
}

function closeModal() {
  document.getElementById('modal-overlay').classList.add('hidden');
}

document.getElementById('modal-overlay').addEventListener('click', (e) => {
  if (e.target === e.currentTarget) closeModal();
});

// ===== Init =====
(async function init() {
  if (API.token) {
    const res = await API.getUserInfo();
    if (res?.success) {
      document.getElementById('username-display').textContent = res.data.username;
      showApp();
      return;
    }
  }
  showLogin();
})();
