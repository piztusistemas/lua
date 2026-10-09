// el: constrúe un nodo DOM dende unha descrición curta. As claves especiais
// son `class`, `text` (textContent), `html` (innerHTML - só para as ICONAS,
// nunca texto do usuario) e `onXxx` (listeners); o resto van como atributos
// (false/null/undefined omítense, true pon o atributo baleiro).
export function el(etq, atrs = {}, ...fillos) {
  const n = document.createElement(etq)
  for (const [k, v] of Object.entries(atrs)) {
    if (v === undefined || v === null || v === false) continue
    if (k === 'class') n.className = v
    else if (k === 'text') n.textContent = v
    else if (k === 'html') n.innerHTML = v
    else if (k.startsWith('on')) n.addEventListener(k.slice(2), v)
    else n.setAttribute(k, v === true ? '' : v)
  }
  for (const f of fillos.flat()) if (f) n.append(f)
  return n
}
