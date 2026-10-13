### Index tag fields - 2026-10-01
- **Context:** Building the index badges from `PropertyType.Label()`.
- **Mistake:** Stored the same string in `indexTag.Class` and `indexTag.Label` in case the CSS class and the visible text might differ later. They do not.
- **Fix/Lesson:** Use one field per value the code actually has. Do not add a second field for a split that nothing requires.

### Function names - 2026-10-01
- **Context:** Naming the helper that returns distinct property-type labels for the index badges.
- **Mistake:** Called helpers `indexTags`, `indexNav`, `indexProjects`, and `indexProps`. The name says where the result is shown, not what it returns.
- **Fix/Lesson:** Name the function after the value (`propertyTypeLabels`, `projectLinksHTML`, `projectRows`, `propertyNames`). The field that holds those labels is `Labels`, not the CSS class `Tags`.

### Price captions - 2026-10-12
- **Context:** Label for `BinaryxEstimatedValue`, the value Binaryx said the property will reach after construction.
- **Mistake:** Used "Promised" / "Обіцяно". That names the claim, not the value.
- **Fix/Lesson:** Use "Projected, Binaryx" / "Прогноз, Binaryx". The Go field is `BinaryxProjectedValue`.

### Saved reports - 2026-10-10
- **Context:** Rounding token and USDT amounts to two decimal places.
- **Mistake:** Edited generated files under `reports/` so existing HTML would show the new formatting.
- **Fix/Lesson:** Never update saved reports. Change the generator only. Reports change when they are regenerated.

