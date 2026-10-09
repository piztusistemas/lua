// Logotipos do asistente: o do provedor de IA que hai configurado en
// Opcións (Gemini, Claude, ChatGPT/OpenAI...) ou, se non hai un propio,
// unhas faíscas xenéricas co degradado de "Intelixencia artificial" do resto
// de Lua. Son SVG sinxelos en liña (nunca texto do usuario), a 24x24.

let contador = 0
const idUnico = (base) => `${base}${++contador}`

const FAISCAS = (id) => `<defs><linearGradient id="${id}" x1="3" y1="3" x2="21" y2="21" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="var(--ai-1)"/><stop offset="1" stop-color="var(--ai-2)"/></linearGradient></defs>` +
  `<path fill="url(#${id})" d="M10 3.2c.6 4.3 3.1 6.8 7.4 7.4-4.3.6-6.8 3.1-7.4 7.4-.6-4.3-3.1-6.8-7.4-7.4 4.3-.6 6.8-3.1 7.4-7.4z"/>` +
  `<path fill="url(#${id})" d="M18.2 13.6c.3 2 1.4 3.1 3.4 3.4-2 .3-3.1 1.4-3.4 3.4-.3-2-1.4-3.1-3.4-3.4 2-.3 3.1-1.4 3.4-3.4z"/>`

const GEMINI = (id) => `<defs><linearGradient id="${id}" x1="4" y1="20" x2="20" y2="4" gradientUnits="userSpaceOnUse"><stop offset="0" stop-color="#4285f4"/><stop offset=".55" stop-color="#9b72cb"/><stop offset="1" stop-color="#d96570"/></linearGradient></defs>` +
  `<path fill="url(#${id})" d="M12 1.5c.6 5.6 4.9 9.9 10.5 10.5-5.6.6-9.9 4.9-10.5 10.5C11.4 16.9 7.1 12.6 1.5 12 7.1 11.4 11.4 7.1 12 1.5z"/>`

const RAIOS_CLAUDE = Array.from({ length: 12 }, (_, i) => {
  const ang = (i * 30 * Math.PI) / 180
  const r0 = 3.2
  const r1 = i % 2 === 0 ? 10.6 : 8.2
  const f = (n) => n.toFixed(2)
  return `<path d="M${f(12 + r0 * Math.sin(ang))} ${f(12 - r0 * Math.cos(ang))}L${f(12 + r1 * Math.sin(ang))} ${f(12 - r1 * Math.cos(ang))}"/>`
}).join('')
const CLAUDE = () => `<g stroke="#d97757" stroke-width="2.3" stroke-linecap="round" fill="none">${RAIOS_CLAUDE}</g>`

const OPENAI = () => '<g fill="none" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round" stroke-linecap="round">' +
  '<path d="M12 2.6l7.4 4.3v8.6L12 19.8l-7.4-4.3V6.9z"/><path d="M12 2.6v8.6l7.4 4.3M12 11.2L4.6 15.5M19.4 6.9L12 11.2 4.6 6.9"/></g>'

const PERPLEXITY = () => '<g fill="none" stroke="#20b8cd" stroke-width="1.7" stroke-linejoin="round" stroke-linecap="round">' +
  '<path d="M12 2.5v19M5 6.8l7 5.2 7-5.2M5 6.8v7.6l7 2.3 7-2.3V6.8M5 6.8L12 2.5l7 4.3"/></g>'

const QWEN = () => '<g fill="none" stroke="#6a4cff" stroke-width="2.2" stroke-linecap="round"><circle cx="11.5" cy="11.5" r="6.8"/><path d="M15 15l5.2 5.2"/></g>'

// logoProvedor devolve o SVG (24x24) do preset de IA ("gemini", "anthropic",
// "openai", "qwen", "perplexity", "outro"...).
export function logoProvedor(preset) {
  const id = idUnico('asl')
  let dentro
  switch (preset) {
    case 'gemini': dentro = GEMINI(id); break
    case 'anthropic': dentro = CLAUDE(); break
    case 'openai': dentro = OPENAI(); break
    case 'qwen': dentro = QWEN(); break
    case 'perplexity': dentro = PERPLEXITY(); break
    default: dentro = FAISCAS(id)
  }
  return `<svg viewBox="0 0 24 24" aria-hidden="true" focusable="false">${dentro}</svg>`
}

// nomeProvedor: o nome amosable dun preset.
export function nomeProvedor(preset) {
  return {
    gemini: 'Gemini', anthropic: 'Claude', openai: 'ChatGPT', qwen: 'Qwen', perplexity: 'Perplexity',
  }[preset] || 'IA'
}
