const API = {
  token: localStorage.getItem('o-ui-token') || '',

  async request(method, path, body) {
    const opts = {
      method,
      headers: { 'Content-Type': 'application/json' }
    };
    if (this.token) opts.headers['Authorization'] = 'Bearer ' + this.token;
    if (body) opts.body = JSON.stringify(body);

    const res = await fetch('/api' + path, opts);
    if (res.status === 401) {
      this.token = '';
      localStorage.removeItem('o-ui-token');
      showLogin();
      return null;
    }
    const data = await res.json();
    return data;
  },

  login(user, pass) { return this.request('POST', '/login', { username: user, password: pass }); },
  getUserInfo() { return this.request('GET', '/user/info'); },
  changePassword(oldP, newP) { return this.request('POST', '/user/password', { old_password: oldP, new_password: newP }); },

  getInbounds() { return this.request('GET', '/inbounds'); },
  addInbound(data) { return this.request('POST', '/inbounds', data); },
  updateInbound(id, data) { return this.request('PUT', '/inbounds/' + id, data); },
  deleteInbound(id) { return this.request('DELETE', '/inbounds/' + id); },

  getOutbounds() { return this.request('GET', '/outbounds'); },
  addOutbound(data) { return this.request('POST', '/outbounds', data); },
  deleteOutbound(id) { return this.request('DELETE', '/outbounds/' + id); },

  getNodes() { return this.request('GET', '/nodes'); },
  addNode(data) { return this.request('POST', '/nodes', data); },
  updateNode(id, data) { return this.request('PUT', '/nodes/' + id, data); },
  deleteNode(id) { return this.request('DELETE', '/nodes/' + id); },
  checkNode(id) { return this.request('POST', '/nodes/' + id + '/check'); },

  getStats() { return this.request('GET', '/stats'); },
  getSystemInfo() { return this.request('GET', '/system'); },
  getCoreStatus() { return this.request('GET', '/core/status'); },
  startCore() { return this.request('POST', '/core/start'); },
  stopCore() { return this.request('POST', '/core/stop'); },
  restartCore() { return this.request('POST', '/core/restart'); },

  getSettings() { return this.request('GET', '/settings'); },
  updateSettings(data) { return this.request('PUT', '/settings', data); },
};
