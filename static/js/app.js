// SplitEase - Frontend SPA
let currentGroup = null;
let groupMembers = [];
let allMembers = [];
let expensePage = 1;
let pendingSettlement = null;

const EPSILON = 1e-9;

function roundMoney(amount) {
  return Math.round(amount * 100) / 100;
}

function moneyEqual(a, b) {
  return Math.abs(a - b) < EPSILON;
}

function moneyLessThan(a, b) {
  return a < b - EPSILON;
}

function moneyGreaterThan(a, b) {
  return a > b + EPSILON;
}

function moneyLessOrEqual(a, b) {
  return a <= b + EPSILON;
}

function moneyGreaterOrEqual(a, b) {
  return a >= b - EPSILON;
}

function navigate(path) {
  history.pushState(null, '', path);
  route();
}

window.addEventListener('popstate', route);

function route() {
  const path = location.pathname;
  document.querySelectorAll('.page').forEach(p => p.style.display = 'none');

  if (path.startsWith('/group/')) {
    const id = path.split('/')[2];
    loadGroup(id);
  } else {
    document.getElementById('page-home').style.display = 'block';
    loadGroups();
  }
}

async function api(path, opts = {}) {
  const res = await fetch('/api' + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
    body: opts.body ? JSON.stringify(opts.body) : undefined,
  });
  if (res.status === 204) return null;
  const text = await res.text();
  if (!text) return null;
  try { return JSON.parse(text); } catch { return text; }
}

function toast(msg) {
  const el = document.createElement('div');
  el.className = 'toast';
  el.textContent = msg;
  document.body.appendChild(el);
  setTimeout(() => el.remove(), 2500);
}

function closeModal(id) {
  document.getElementById(id).style.display = 'none';
}

function switchTab(btn) {
  const tabId = btn.dataset.tab;
  btn.parentElement.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  btn.classList.add('active');
  document.querySelectorAll('.tab-content').forEach(t => { t.style.display = 'none'; t.classList.remove('active'); });
  const target = document.getElementById(tabId);
  target.style.display = 'block';
  target.classList.add('active');

  if (tabId === 'tab-balances') loadBalances();
  if (tabId === 'tab-expenses') { expensePage = 1; loadExpenses(); }
  if (tabId === 'tab-settle') loadSettlements();
  if (tabId === 'tab-stats') loadStats();
}

async function loadGroups() {
  const groups = await api('/groups') || [];
  const list = document.getElementById('group-list');
  const empty = document.getElementById('no-groups');

  if (!groups.length) {
    list.innerHTML = '';
    empty.style.display = 'block';
    return;
  }
  empty.style.display = 'none';
  list.innerHTML = groups.map(g => `
    <div class="card" onclick="navigate('/group/${g.id}')">
      <h3>${esc(g.name)}</h3>
      <div class="meta">${esc(g.description || '无描述')} · ${g.currency}</div>
    </div>
  `).join('');
}

function showCreateGroup() {
  document.getElementById('modal-create-group').style.display = 'flex';
}

async function createGroup() {
  const name = document.getElementById('cg-name').value.trim();
  if (!name) return toast('请输入分组名称');
  const desc = document.getElementById('cg-desc').value.trim();
  const currency = document.getElementById('cg-currency').value;
  await api('/groups', { method: 'POST', body: { name, description: desc, currency } });
  closeModal('modal-create-group');
  toast('分组已创建');
  loadGroups();
}

async function loadGroup(id) {
  const group = await api('/groups/' + id);
  if (!group) return navigate('/');
  currentGroup = group;
  groupMembers = group.members || [];

  document.getElementById('page-home').style.display = 'none';
  document.getElementById('page-group').style.display = 'block';
  document.getElementById('group-name').textContent = group.name;
  document.getElementById('group-currency').textContent = group.currency;

  await loadAllMembers();
  renderMembers();
  loadExpenses();
}

async function loadAllMembers() {
  allMembers = await api('/members') || [];
  const sel = document.getElementById('member-select');
  sel.innerHTML = '<option value="">选择已有成员...</option>' +
    allMembers.map(m => `<option value="${m.id}">${esc(m.nickname)}</option>`).join('');
}

function renderMembers() {
  const container = document.getElementById('member-list');
  container.innerHTML = groupMembers.map(m => `
    <span class="member-chip">${esc(m.nickname)}<span class="remove" onclick="removeMember(${m.id})">×</span></span>
  `).join('');
}

async function addMemberToGroup() {
  const sel = document.getElementById('member-select');
  const memberId = sel.value;
  if (!memberId) return toast('请选择一个成员');
  await api(`/groups/${currentGroup.id}/members`, { method: 'POST', body: { member_id: Number(memberId) } });
  toast('已添加');
  loadGroup(currentGroup.id);
}

async function createAndAddMember() {
  const name = document.getElementById('new-member-name').value.trim();
  if (!name) return toast('请输入昵称');
  const member = await api('/members', { method: 'POST', body: { nickname: name } });
  if (!member) return toast('创建失败，昵称可能已存在');
  await api(`/groups/${currentGroup.id}/members`, { method: 'POST', body: { member_id: member.id } });
  document.getElementById('new-member-name').value = '';
  toast('已创建并添加');
  loadGroup(currentGroup.id);
}

async function removeMember(memberId) {
  if (!confirm('确定移除该成员？')) return;
  const result = await api(`/groups/${currentGroup.id}/members/${memberId}`, { method: 'DELETE' });
  if (result && result.error) {
    toast(result.error);
  } else {
    loadGroup(currentGroup.id);
  }
}

async function loadExpenses() {
  const expenses = await api(`/groups/${currentGroup.id}/expenses?page=${expensePage}`) || [];
  const list = document.getElementById('expense-list');

  if (!expenses.length && expensePage === 1) {
    list.innerHTML = '<div class="empty-state">暂无支出记录</div>';
    return;
  }

  list.innerHTML = expenses.map(e => `
    <div class="expense-item">
      <div class="info">
        <div class="desc">${esc(e.description || '无备注')}</div>
        <div class="meta">${esc(e.payer_name)} 付款 · ${e.expense_date} · ${e.split_type}分摊</div>
      </div>
      <div class="amount">${e.currency} ${Number(e.amount).toFixed(2)}</div>
      <div class="actions">
        <button class="btn btn-sm btn-danger" onclick="deleteExpense(${e.id})">删除</button>
      </div>
    </div>
  `).join('');

  document.getElementById('expense-pagination').innerHTML = expenses.length >= 20 ?
    `<button class="btn btn-sm" onclick="expensePage++;loadExpenses()">下一页</button>` : '';
}

function showAddExpense() {
  if (!groupMembers.length) return toast('请先添加成员');
  const payerSel = document.getElementById('ex-payer');
  payerSel.innerHTML = groupMembers.map(m => `<option value="${m.id}">${esc(m.nickname)}</option>`).join('');
  document.getElementById('ex-date').value = new Date().toISOString().split('T')[0];
  document.getElementById('ex-amount').value = '';
  document.getElementById('ex-desc').value = '';
  document.getElementById('ex-rate').value = '1';
  document.getElementById('split-validation').style.display = 'none';
  updateSplitUI();
  document.getElementById('modal-add-expense').style.display = 'flex';
}

function updateSplitUI() {
  const type = document.getElementById('ex-split-type').value;
  const container = document.getElementById('split-members');
  document.getElementById('split-validation').style.display = 'none';

  if (type === 'equal') {
    container.innerHTML = '<div style="font-size:13px;color:#888;margin-top:8px">所有成员均摊</div>';
    return;
  }

  container.innerHTML = groupMembers.map(m => `
    <div class="split-row">
      <input type="checkbox" checked data-member="${m.id}" onchange="validateSplits()">
      <span class="name">${esc(m.nickname)}</span>
      <input type="number" step="0.01" placeholder="${type === 'percentage' ? '%' : '金额'}" data-member-amount="${m.id}" oninput="validateSplits()">
    </div>
  `).join('');
}

function validateSplits() {
  const splitType = document.getElementById('ex-split-type').value;
  const validationEl = document.getElementById('split-validation');
  
  if (splitType === 'equal') {
    validationEl.style.display = 'none';
    return true;
  }

  const checkboxes = document.querySelectorAll('#split-members input[type="checkbox"]:checked');
  if (checkboxes.length === 0) {
    validationEl.textContent = '请选择至少一个参与分摊的成员';
    validationEl.style.display = 'block';
    return false;
  }

  const totalAmount = parseFloat(document.getElementById('ex-amount').value) || 0;
  
  if (splitType === 'percentage') {
    let totalPercent = 0;
    let hasNegative = false;
    checkboxes.forEach(cb => {
      const mid = Number(cb.dataset.member);
      const inp = document.querySelector(`[data-member-amount="${mid}"]`);
      const val = parseFloat(inp.value) || 0;
      if (moneyLessThan(val, 0)) hasNegative = true;
      totalPercent = roundMoney(totalPercent + val);
    });
    
    if (hasNegative) {
      validationEl.textContent = '百分比不能为负数';
      validationEl.style.display = 'block';
      return false;
    }
    
    if (!moneyEqual(totalPercent, 100)) {
      const diff = roundMoney(100 - totalPercent);
      validationEl.textContent = `百分比总和为 ${totalPercent.toFixed(2)}%，需等于 100%（相差 ${Math.abs(diff).toFixed(2)}%）`;
      validationEl.style.display = 'block';
      return false;
    }
  } else if (splitType === 'exact') {
    let totalSplit = 0;
    let hasNegative = false;
    checkboxes.forEach(cb => {
      const mid = Number(cb.dataset.member);
      const inp = document.querySelector(`[data-member-amount="${mid}"]`);
      const val = parseFloat(inp.value) || 0;
      if (moneyLessThan(val, 0)) hasNegative = true;
      totalSplit = roundMoney(totalSplit + val);
    });
    
    if (hasNegative) {
      validationEl.textContent = '分摊金额不能为负数';
      validationEl.style.display = 'block';
      return false;
    }
    
    if (!moneyEqual(totalSplit, totalAmount)) {
      const diff = roundMoney(totalAmount - totalSplit);
      validationEl.textContent = `分摊金额总和为 ${totalSplit.toFixed(2)}，需等于支出总额 ${totalAmount.toFixed(2)}（相差 ${Math.abs(diff).toFixed(2)}）`;
      validationEl.style.display = 'block';
      return false;
    }
  }

  validationEl.style.display = 'none';
  return true;
}

async function submitExpense() {
  const amount = parseFloat(document.getElementById('ex-amount').value);
  const payerId = Number(document.getElementById('ex-payer').value);
  const currency = document.getElementById('ex-currency').value || undefined;
  const rate = parseFloat(document.getElementById('ex-rate').value) || 1;
  const desc = document.getElementById('ex-desc').value.trim();
  const date = document.getElementById('ex-date').value;
  const splitType = document.getElementById('ex-split-type').value;
  const validationEl = document.getElementById('split-validation');

  if (!amount || !payerId) {
    validationEl.textContent = '请填写金额和付款人';
    validationEl.style.display = 'block';
    return;
  }
  if (moneyLessThan(amount, 0)) {
    validationEl.textContent = '金额不能为负数';
    validationEl.style.display = 'block';
    return;
  }

  if (!validateSplits()) {
    return;
  }

  let splits = [];
  if (splitType === 'equal') {
    splits = groupMembers.map(m => ({ member_id: m.id }));
  } else {
    const checkboxes = document.querySelectorAll('#split-members input[type="checkbox"]:checked');
    checkboxes.forEach(cb => {
      const mid = Number(cb.dataset.member);
      const inp = document.querySelector(`[data-member-amount="${mid}"]`);
      const val = parseFloat(inp.value) || 0;
      splits.push({
        member_id: mid,
        amount: val,
        percentage: val,
      });
    });
  }

  if (!splits.length) {
    validationEl.textContent = '请选择至少一个参与分摊的成员';
    validationEl.style.display = 'block';
    return;
  }

  const result = await api(`/groups/${currentGroup.id}/expenses`, {
    method: 'POST',
    body: { payer_id: payerId, amount, currency, exchange_rate: rate, description: desc, split_type: splitType, expense_date: date, splits },
  });

  if (result && result.error) {
    const validationEl = document.getElementById('split-validation');
    let errorMsg = result.error;
    
    if (result.diff !== undefined) {
      const absDiff = Math.abs(result.diff);
      if (result.split_type === 'percentage') {
        const total = result.total_percent !== undefined ? result.total_percent.toFixed(2) : '?';
        errorMsg = `百分比总和为 ${total}%，需等于 100%（相差 ${absDiff.toFixed(2)}%）`;
      } else if (result.split_type === 'exact') {
        const total = result.total_amount !== undefined ? result.total_amount.toFixed(2) : '?';
        const expected = result.expected_total !== undefined ? result.expected_total.toFixed(2) : '?';
        errorMsg = `分摊金额总和为 ${total}，需等于支出总额 ${expected}（相差 ${absDiff.toFixed(2)}）`;
      }
    }
    
    validationEl.textContent = errorMsg;
    validationEl.style.display = 'block';
    return;
  }

  closeModal('modal-add-expense');
  toast('支出已记录');
  loadExpenses();
}

async function deleteExpense(id) {
  if (!confirm('确定删除这笔支出？')) return;
  await api(`/expenses/${id}`, { method: 'DELETE' });
  toast('已删除');
  loadExpenses();
}

async function loadBalances() {
  const balances = await api(`/groups/${currentGroup.id}/balances`) || [];
  const container = document.getElementById('balance-list');

  container.innerHTML = balances.map(b => {
    const cls = moneyGreaterThan(b.amount, 0.01) ? 'balance-positive' : 
                moneyLessThan(b.amount, -0.01) ? 'balance-negative' : 'balance-zero';
    const label = moneyGreaterThan(b.amount, 0.01) ? '应收' : 
                  moneyLessThan(b.amount, -0.01) ? '应付' : '已清';
    return `
      <div class="balance-item">
        <span class="name">${esc(b.member_name)}</span>
        <span class="amount ${cls}">${label} ${Math.abs(b.amount).toFixed(2)}</span>
      </div>
    `;
  }).join('');
}

async function loadSuggestions() {
  const suggestions = await api(`/groups/${currentGroup.id}/settlements/suggest`) || [];
  const container = document.getElementById('settlement-suggestions');

  if (!suggestions.length) {
    container.innerHTML = '<div class="empty-state">所有账目已结清 🎉</div>';
    return;
  }

  container.innerHTML = '<h3>建议结算方案</h3>' + suggestions.map((s, i) => `
    <div class="settle-item">
      <div class="flow">
        <strong>${esc(s.from.nickname)}</strong>
        <span class="arrow">→</span>
        <strong>${esc(s.to.nickname)}</strong>
      </div>
      <div class="amount">${s.amount.toFixed(2)}</div>
      <button class="btn btn-sm btn-primary" onclick="showSettlementConfirm(${s.from.id},${s.to.id},${s.amount})">确认付款</button>
    </div>
  `).join('');
}

async function showSettlementConfirm(payerId, payeeId, amount) {
  const balances = await api(`/groups/${currentGroup.id}/balances`) || [];
  
  let payerBalance = 0, payeeBalance = 0;
  let payerName = '', payeeName = '';
  
  for (const b of balances) {
    if (b.member_id === payerId) {
      payerBalance = b.amount;
      payerName = b.member_name;
    }
    if (b.member_id === payeeId) {
      payeeBalance = b.amount;
      payeeName = b.member_name;
    }
  }
  
  const maxCanPay = Math.min(-payerBalance, payeeBalance);
  
  pendingSettlement = { payerId, payeeId, amount, payerName, payeeName, payerBalance, payeeBalance, maxCanPay };
  
  const content = document.getElementById('settle-confirm-content');
  content.innerHTML = `
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
      <div style="text-align:center">
        <div style="font-weight:bold;font-size:16px">${esc(payerName)}</div>
        <div style="color:#dc3545;margin-top:4px">应付 ${(-payerBalance).toFixed(2)}</div>
      </div>
      <div style="font-size:24px;color:#666">→</div>
      <div style="text-align:center">
        <div style="font-weight:bold;font-size:16px">${esc(payeeName)}</div>
        <div style="color:#28a745;margin-top:4px">应收 ${payeeBalance.toFixed(2)}</div>
      </div>
    </div>
    <div style="background:#f8f9fa;padding:12px;border-radius:8px;margin-bottom:12px">
      <div style="font-size:13px;color:#666;margin-bottom:4px">结算金额</div>
      <div style="display:flex;align-items:center;gap:8px">
        <input type="number" id="settle-amount-input" step="0.01" value="${amount.toFixed(2)}" 
               style="flex:1;padding:8px;border:1px solid #ddd;border-radius:4px"
               oninput="updateSettlementAmount()">
      </div>
    </div>
    <div id="settle-amount-validation" style="color:#dc3545;font-size:13px;display:none"></div>
    <div style="font-size:12px;color:#888;margin-top:8px">
      最大可结算金额：<strong>${maxCanPay.toFixed(2)}</strong>
    </div>
  `;
  
  const confirmBtn = document.getElementById('btn-settle-confirm');
  confirmBtn.onclick = executeSettlement;
  
  document.getElementById('modal-settle-confirm').style.display = 'flex';
}

function updateSettlementAmount() {
  const input = document.getElementById('settle-amount-input');
  const validation = document.getElementById('settle-amount-validation');
  const amount = parseFloat(input.value) || 0;
  
  if (!pendingSettlement) return;
  
  if (moneyLessThan(amount, 0)) {
    validation.textContent = '结算金额不能为负数';
    validation.style.display = 'block';
    return;
  }
  
  if (moneyGreaterThan(amount, pendingSettlement.maxCanPay)) {
    validation.textContent = `结算金额不能超过最大可结算金额 ${pendingSettlement.maxCanPay.toFixed(2)}`;
    validation.style.display = 'block';
    return;
  }
  
  validation.style.display = 'none';
}

async function executeSettlement() {
  if (!pendingSettlement) return;
  
  const input = document.getElementById('settle-amount-input');
  const amount = parseFloat(input.value) || 0;
  
  if (moneyLessOrEqual(amount, 0)) {
    toast('请输入有效的结算金额');
    return;
  }
  
  if (moneyGreaterThan(amount, pendingSettlement.maxCanPay)) {
    toast(`结算金额不能超过 ${pendingSettlement.maxCanPay.toFixed(2)}`);
    return;
  }
  
  const result = await api(`/groups/${currentGroup.id}/settlements`, {
    method: 'POST',
    body: { payer_id: pendingSettlement.payerId, payee_id: pendingSettlement.payeeId, amount },
  });
  
  if (result && result.error) {
    toast(result.error);
    return;
  }
  
  closeModal('modal-settle-confirm');
  pendingSettlement = null;
  toast('结算已记录');
  loadSuggestions();
  loadSettlements();
}

async function loadSettlements() {
  const settlements = await api(`/groups/${currentGroup.id}/settlements`) || [];
  const container = document.getElementById('settlement-history');

  if (!settlements.length) {
    container.innerHTML = '<div class="empty-state">暂无结算记录</div>';
    return;
  }

  container.innerHTML = settlements.map(s => `
    <div class="settle-item">
      <div class="flow">
        <strong>${esc(s.payer_name)}</strong>
        <span class="arrow">→</span>
        <strong>${esc(s.payee_name)}</strong>
      </div>
      <div class="amount">${s.amount.toFixed(2)}</div>
      <div class="meta" style="font-size:12px;color:#888">${s.created_at}</div>
    </div>
  `).join('');
}

async function loadStats() {
  const stats = await api(`/groups/${currentGroup.id}/stats`);
  if (!stats) return;

  const container = document.getElementById('stats-content');
  let html = `
    <div class="stats-grid">
      <div class="stat-card">
        <div class="label">总支出</div>
        <div class="value">${stats.total_expenses.toFixed(2)}</div>
      </div>
      <div class="stat-card">
        <div class="label">支出笔数</div>
        <div class="value">${stats.expense_count}</div>
      </div>
    </div>
  `;

  if (stats.member_stats && stats.member_stats.length) {
    html += '<h3 style="margin-bottom:12px">成员明细</h3>';
    html += stats.member_stats.map(ms => {
      const cls = moneyGreaterThan(ms.net_balance, 0) ? 'balance-positive' : 
                  moneyLessThan(ms.net_balance, 0) ? 'balance-negative' : 'balance-zero';
      return `
        <div class="balance-item">
          <span class="name">${esc(ms.member_name)}</span>
          <span style="font-size:13px;color:#888">付 ${ms.total_paid.toFixed(2)} · 分摊 ${ms.total_owed.toFixed(2)}</span>
          <span class="amount ${cls}">${ms.net_balance.toFixed(2)}</span>
        </div>
      `;
    }).join('');
  }

  container.innerHTML = html;
}

function esc(s) {
  if (!s) return '';
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

document.addEventListener('DOMContentLoaded', () => {
  route();
  document.getElementById('ex-date').valueAsDate = new Date();
});
