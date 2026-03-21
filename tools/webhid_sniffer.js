/**
 * Fractal Scape WebHID Protocol Sniffer
 * ======================================
 *
 * USAGE:
 * 1. Open https://adjust.fractal-design.com in Chrome
 * 2. Open DevTools (F12) → Console tab
 * 3. Paste this ENTIRE script and press Enter
 * 4. NOW click "Add Fractal USB Device" in the web app
 * 5. Exercise each feature (change EQ, RGB, check battery, etc.)
 * 6. Watch the console for logged HID reports
 *
 * HELPERS:
 *   scapeNote("changed EQ band 1 to +3dB")  — annotate the log
 *   scapeExport()                            — download full log as JSON
 *   copy(JSON.stringify(__SCAPE_LOG, null, 2)) — copy to clipboard
 */
(() => {
  'use strict';
  window.__SCAPE_LOG = [];

  const ts = () => new Date().toISOString();
  const hex = (arr) => [...new Uint8Array(arr)].map(b => b.toString(16).padStart(2, '0')).join(' ');
  const log = (entry) => {
    window.__SCAPE_LOG.push(entry);
    console.log(`%c[SCAPE ${entry.type}]`, 'color: #00ff88; font-weight: bold', entry);
  };

  // ── Intercept open (captures device info + report descriptors) ──
  const origOpen = HIDDevice.prototype.open;
  HIDDevice.prototype.open = async function () {
    const result = await origOpen.call(this);
    log({
      type: 'DEVICE_OPENED', ts: ts(),
      vendorId: '0x' + this.vendorId.toString(16).padStart(4, '0'),
      productId: '0x' + this.productId.toString(16).padStart(4, '0'),
      productName: this.productName,
      collections: JSON.parse(JSON.stringify(this.collections)),
    });
    console.log('%c[SCAPE] Collections (report descriptors):', 'color: #ff8800; font-weight: bold');
    for (const col of this.collections) {
      console.log(`  UsagePage: 0x${col.usagePage.toString(16)} Usage: 0x${col.usage.toString(16)}`);
      for (const r of (col.inputReports || []))
        console.log(`    Input  ID: ${r.reportId}`, r.items);
      for (const r of (col.outputReports || []))
        console.log(`    Output ID: ${r.reportId}`, r.items);
      for (const r of (col.featureReports || []))
        console.log(`    Feature ID: ${r.reportId}`, r.items);
    }
    // Capture incoming reports
    this.addEventListener('inputreport', (e) => {
      const d = new Uint8Array(e.data.buffer);
      log({ type: 'RX_INPUT', ts: ts(), reportId: e.reportId,
             dataHex: hex(d), dataArray: [...d], length: d.length });
    });
    return result;
  };

  // ── Intercept outgoing output reports ──
  const origSend = HIDDevice.prototype.sendReport;
  HIDDevice.prototype.sendReport = function (id, data) {
    const a = new Uint8Array(data);
    log({ type: 'TX_OUTPUT', ts: ts(), reportId: id,
           dataHex: hex(a), dataArray: [...a], length: a.length });
    return origSend.call(this, id, data);
  };

  // ── Intercept outgoing feature reports ──
  const origSendFeat = HIDDevice.prototype.sendFeatureReport;
  HIDDevice.prototype.sendFeatureReport = function (id, data) {
    const a = new Uint8Array(data);
    log({ type: 'TX_FEATURE', ts: ts(), reportId: id,
           dataHex: hex(a), dataArray: [...a], length: a.length });
    return origSendFeat.call(this, id, data);
  };

  // ── Intercept incoming feature reports ──
  const origRecvFeat = HIDDevice.prototype.receiveFeatureReport;
  HIDDevice.prototype.receiveFeatureReport = async function (id) {
    const result = await origRecvFeat.call(this, id);
    const d = new Uint8Array(result.buffer);
    log({ type: 'RX_FEATURE', ts: ts(), reportId: id,
           dataHex: hex(d), dataArray: [...d], length: d.length });
    return result;
  };

  // ── Intercept requestDevice (logs VID/PID filters) ──
  const origReq = navigator.hid.requestDevice.bind(navigator.hid);
  navigator.hid.requestDevice = async function (opts) {
    log({ type: 'REQUEST_DEVICE', ts: ts(), filters: opts?.filters });
    return origReq(opts);
  };

  // ── Helpers ──
  window.scapeNote = (note) => {
    log({ type: 'USER_NOTE', ts: ts(), note });
    console.log(`%c[SCAPE] Note: ${note}`, 'color: #ffff00');
  };
  window.scapeExport = () => {
    const blob = new Blob([JSON.stringify(window.__SCAPE_LOG, null, 2)], { type: 'application/json' });
    const a = document.createElement('a'); a.href = URL.createObjectURL(blob);
    a.download = `scape-hid-log-${Date.now()}.json`; a.click();
    console.log(`%c[SCAPE] Exported ${window.__SCAPE_LOG.length} entries`, 'color: #00ff88');
  };

  console.log('%c[SCAPE SNIFFER ACTIVE]', 'color: #00ff88; font-size: 16px; font-weight: bold');
  console.log('%cNow connect your device via the web app.', 'color: #aaa');
  console.log('  scapeNote("description") — annotate | scapeExport() — download log');
})();
