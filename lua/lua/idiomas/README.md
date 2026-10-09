# Idiomas de Lua

`gl.json` é a fonte da verdade: contén todas as cadeas de texto da interface
de Lua, en galego, cunha clave por cadea (`"toolbar.new.label": "Novo"`).

`es.json`, `en.json` e `pt.json` están traducidos por completo (as mesmas
claves que `gl.json`). Se algún día falta unha clave, non rompe nada: o
dicionario do idioma activo cae automaticamente a `gl.json` para calquera
clave que aínda non teña tradución (ver `t()` en `frontend/src/main.js`), así
que Lua funciona igual en calquera idioma dende o primeiro momento — só se
ve en galego a cadea que falte.

## Como traducir

1. Copia todo o contido de `gl.json`.
2. Pídelle a outra IA que traduza os **valores** ao castelán/inglés/portugués,
   mantendo as **claves** (a parte antes dos dous puntos) exactamente igual —
   son o que o programa busca, non se amosan nunca.
3. Respecta os `{marcadores}` tal cal (ex. `{n}`, `{msg}`, `{path}`): Lua
   substitúeos en tempo real por un número, unha mensaxe de erro, etc. Non
   traducir o texto dentro das chaves, só o texto ao redor.
4. As claves que rematan en `.html` (ex. `latexManual.other.html`) levan
   marcado HTML dentro (normalmente un `<a href="...">`); traduce só o
   texto visible, deixa as etiquetas tal cal.
5. Pega o resultado no ficheiro correspondente (`es.json`, `en.json` ou
   `pt.json`), substituíndo o `{}` que hai agora.
6. Recompila o frontend (`npm run build` dentro de `lua/frontend/`) e
   `wails build` — os JSON impórtanse en tempo de compilación, non se len en
   tempo de execución.

O idioma escóllese en Lua → Opcións → Idioma, e gárdase nos axustes do
usuario (aplícase por completo ao reiniciar Lua).
