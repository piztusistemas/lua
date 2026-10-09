---
luayang: 1
tipo: exercicio
titulo: Sentence transformation
materia: Inglés
nivel: 2º Bacharelato
conceptos: ["[[Passive voice]]", "[[Reported speech]]", "[[Present perfect]]"]
fonte:
  pdf: "[[inglesex4rev.pdf]]"
  paxina: 2
  apartado: "2.1"
puntos: 1.5
linguas: [en]
campos:
  # A variante de letras: o azar escolle filas dun banco.
  # O banco énchese a man, cun CSV ou coa IA (verificado e revisado);
  # unha vez gardado, é fixo: a mesma semente dá sempre o mesmo exame.
  - nome: frases
    orixe: taboa
    banco: "[[banco-transformacions]]"
    mostra: 6
    distintas_por: estrutura   # non repetir dúas da mesma estrutura
probas:
  sementes: 25
---

## Enunciado

Write a NEW sentence that has the same meaning as the given one. Use the word
or expression in brackets.

{frases | lista: "{orixinal} (*{pista}*)"}

## Resultado

{frases | lista: "{solucion}"}
