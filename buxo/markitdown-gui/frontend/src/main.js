import './style.css';
import { SelectPDFs, SelectOutputDir, Convert } from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

const $ = (id) => document.getElementById(id);
const state = { left: [], right: [], sel: new Set(), outDir: '' };
const base = (p) => p.split(/[\\/]/).pop();
const dir = (p) => p.slice(0, p.length - base(p).length - 1);
const status = (t) => ($('status').textContent = t);

function render() {
  for (const side of ['left', 'right']) {
    const ul = $(side);
    ul.replaceChildren(...state[side].map((p) => {
      const li = document.createElement('li');
      li.className = state.sel.has(p) ? 'sel' : '';
      li.dataset.path = p;
      li.innerHTML = '<span></span><small></small>';
      li.firstChild.textContent = base(p);
      li.lastChild.textContent = dir(p);
      li.title = p;
      li.onclick = () => { state.sel.has(p) ? state.sel.delete(p) : state.sel.add(p); render(); };
      return li;
    }));
  }
  $('outDir').value = state.outDir;
  $('convert').disabled = !state.right.length || !state.outDir;
}

function move(from, to) {
  const picked = state[from].filter((p) => state.sel.has(p));
  const items = picked.length ? picked : [...state[from]];
  for (const p of items) {
    state[from] = state[from].filter((x) => x !== p);
    if (!state[to].includes(p)) state[to].push(p);
    state.sel.delete(p);
  }
  if (from === 'left' && !state.outDir && state.right.length) state.outDir = dir(state.right[0]);
  render();
}

$('add').onclick = async () => {
  const files = (await SelectPDFs()) || [];
  for (const f of files) if (!state.left.includes(f) && !state.right.includes(f)) state.left.push(f);
  status(`${state.left.length} ficheiro(s) dispoñible(s).`);
  render();
};
$('clearLeft').onclick = () => { state.left = []; render(); };
$('toRight').onclick = () => move('left', 'right');
$('toLeft').onclick = () => move('right', 'left');
$('pickOut').onclick = async () => {
  const d = await SelectOutputDir(state.outDir);
  if (d) { state.outDir = d; render(); }
};

EventsOn('progress', (r) => {
  const li = [...$('right').children].find((e) => e.dataset.path === r.input);
  if (li) li.classList.add(r.ok ? 'ok' : 'err');
  if (li && !r.ok) li.title = r.error;
  status(r.ok ? `Convertido: ${base(r.output)}` : `Erro en ${base(r.input)}: ${r.error}`);
});

$('convert').onclick = async () => {
  $('convert').disabled = true;
  status('Convertendo… (a primeira vez pode tardar uns segundos)');
  try {
    const res = (await Convert(state.right, state.outDir)) || [];
    const ok = res.filter((r) => r.ok).length;
    status(`Rematado: ${ok} de ${state.right.length} convertidos en ${state.outDir}`);
  } catch (e) {
    status('Erro: ' + e);
  }
  $('convert').disabled = false;
};

render();
