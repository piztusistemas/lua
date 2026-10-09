---
luayang: 1
tipo: exercicio
titulo: Área entre unha parábola e o eixe x
materia: Matemáticas II
nivel: 2º Bacharelato
conceptos: ["[[Integral definida]]", "[[Área dunha rexión plana]]"]
fonte:
  pdf: "[[matematicasiiex426.pdf]]"
  paxina: 2
  apartado: "3.3"
puntos: 2
linguas: [gl]
campos:
  # O truco de Yang: sortéase a SOLUCIÓN e calcúlase o DATO.
  # Así a resposta é sempre «bonita» e nunca hai que resolver ao revés.
  - nome: solA
    orixe: aleatorio
    xerador: rango
    de: 1
    ata: 6
  - nome: k
    orixe: aleatorio
    xerador: lista
    valores: [1, 2, 3, 4]
  - nome: S
    orixe: calculado
    motor: maxima
    expr: "solA*k^3/6"
    formato: latex
  - nome: f
    orixe: calculado
    motor: maxima
    expr: "a*x^2 - k*a*x"
    formato: latex
probas:
  sementes: 25
  condicions: ["S > 0"]
---

## Enunciado

Calcule o valor que ten que tomar o parámetro positivo $a$ para que a área da
rexión encerrada pola gráfica da función $f(x)={f}$ e o eixe $x$ sexa igual a
${S}$ unidades de superficie. Faga un debuxo da rexión.

## Resultado

$a = {solA}$

## Resolución

A parábola corta o eixe $x$ en $x=0$ e en $x={k}$. Como $a>0$, entre eses
puntos a función é negativa, así que

$$
\text{Área} = \int_0^{{k}} -f(x)\,dx = \frac{a\cdot {k}^3}{6}.
$$

Igualando á área pedida, $\dfrac{a\cdot {k}^3}{6} = {S}$, de onde $a = {solA}$.

```plot
plot2d(subst(a=solA, f), [x, -0.5, k+0.5])
```
