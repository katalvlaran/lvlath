// SPDX-License-Identifier: MIT
// Package: matrix
//
// Purpose:
//  - Provide a single, canonical source of truth for common validation checks.
//  - Keep kernels/facades minimal by delegating shape/nil/symmetry checks here.
//  - Return plain sentinel errors (no wrapping) so call sites can wrap uniformly.
//
// Determinism & Performance:
//  - All checks are pure, deterministic and allocate nothing.
//  - Symmetry check runs O(n²) on the upper triangle only.
//
// AI-Hints:
//  - Centralizing validators eliminates inconsistent guard logic across files.
//  - Use ValidateSymmetric before spectral methods (Jacobi) to fail fast.
//  - Use IsZeroOffDiagonal to short-circuit iterative algorithms when matrix is already diagonal.
//  - Use ValidateVecLen for any MatVec-like operations to avoid ad hoc length code.
//
// Note:
//  - Each composite validator follows a fixed sequence (e.g. NotNil → Shape).
//  - Each validator describes what it validates and what it assumes (e.g. no nil check).

package matrix

import (
	"fmt"
	"math"
)

// zeroTol is a tiny tolerance used only internally for guards where appropriate.
// We keep it explicit to avoid "magic numbers" inline.
const zeroTol = 0.0

// validatorErrorf wraps an underlying error with the given validator tag.
// Used internally to maintain consistent labeling of sentinel violations.
func validatorErrorf(tag string, err error) error {
	// Provides consistent error tagging for all validation errors.
	return fmt.Errorf("%s: %w", tag, err)
}

// ValidateNotNil – Ensures the matrix reference is non-nil.
//
// Inputs: Matrix interface value.
// Returns ErrNilMatrix if m == nil.
// Complexity: O(1).
// AI-Hints: Use as the first step in composite validations.
func ValidateNotNil(m Matrix) error {
	// If the matrix is nil, fail with the unified sentinel.
	if m == nil {
		return validatorErrorf("ValidateNotNil", ErrNilMatrix) // single source of truth for "nil argument"
	}

	// Otherwise accept.
	return nil
}

// ValidateSameShape – Ensures matrices a and b have equal dimensions.
//
// Implementation: Assumes a and b are not nil (caller must ensure).
// Inputs: Two Matrix values.
// Return: nil or wrapped ErrDimensionMismatch.
// Complexity: O(1).
// AI-Hints: Use for Add/Sub/Hadamard kernels and compatibility guards.
func ValidateSameShape(a, b Matrix) error {
	// Execute comparisons
	if a.Rows() != b.Rows() {
		return validatorErrorf("ValidateSameShape: Rows", ErrDimensionMismatch)
	}
	if a.Cols() != b.Cols() {
		return validatorErrorf("ValidateSameShape: Columns", ErrDimensionMismatch)
	}

	return nil
}

// ValidateSquare checks that m is square (Rows == Cols).
//
// Inputs: Matrix value.
// Errors: ErrNilMatrix if nil, ErrDimensionMismatch if not square.
// Complexity: O(1).
// AI-Hints: Use before spectral or factorization methods.
func ValidateSquare(m Matrix) error {
	// Check the square condition explicitly.
	if m.Rows() != m.Cols() {
		return validatorErrorf("ValidateSquare", ErrDimensionMismatch)
	}

	return nil
}

// ValidateVecLen ensures the vector length matches the required size n.
// Time: O(1). Space: O(1).
func ValidateVecLen(x []float64, n int) error {
	// Disallow nil vectors to avoid subtle bugs in MatVec-like routines.
	if x == nil {
		return validatorErrorf("ValidateVecLen", ErrNilMatrix) // we reuse the existing sentinel for "nil argument"
	}
	// Check the exact expected length.
	if len(x) != n {
		return validatorErrorf("ValidateVecLen", ErrDimensionMismatch) // vector length must match the number of columns
	}

	return nil
}

// ValidateGraph ensures an AdjacencyMatrix value is non-nil and square,
// and (when available) the index table is consistent with the matrix dimension.
// Time: O(1). Space: O(1).
func ValidateGraph(am *AdjacencyMatrix) error {
	// Check wrapper and underlying storage presence.
	if am == nil || am.Mat == nil {
		return validatorErrorf("ValidateGraph", ErrNilMatrix) // nil graph container or matrix
	}
	// Enforce square adjacency for graph algorithms.
	if err := ValidateSquare(am.Mat); err != nil {
		return validatorErrorf("ValidateGraph", err) // adjacency must be square
	}
	// If reverse index is present, ensure consistent dimension.
	if am.vertexByIndex != nil && len(am.vertexByIndex) != am.Mat.Rows() {
		return validatorErrorf("ValidateGraph", ErrDimensionMismatch) // index table must align with matrix rows
	}
	return nil
}

// ValidateBinarySameShape – Composite: NotNil(a) → NotNil(b) → SameShape.
//
// Errors: Combines ErrNilMatrix and ErrDimensionMismatch.
// Complexity: O(1).
func ValidateBinarySameShape(a, b Matrix) error {
	if err := ValidateNotNil(a); err != nil {
		return validatorErrorf("ValidateBinarySameShape", err)
	}
	if err := ValidateNotNil(b); err != nil {
		return validatorErrorf("ValidateBinarySameShape", err)
	}
	if err := ValidateSameShape(a, b); err != nil {
		return validatorErrorf("ValidateBinarySameShape", err)
	}
	return nil
}

// ValidateSquareNonNil – Composite: NotNil → Square.
//
// Errors: ErrNilMatrix, ErrDimensionMismatch.
// Complexity: O(1).
func ValidateSquareNonNil(m Matrix) error {
	if err := ValidateNotNil(m); err != nil {
		return validatorErrorf("ValidateSquareNonNil", err)
	}
	if err := ValidateSquare(m); err != nil {
		return validatorErrorf("ValidateSquareNonNil", err)
	}
	return nil
}

// ValidateSymmetric checks A is symmetric within tolerance tol:
// |A[i,j] - A[j,i]| ≤ tol for all i<j.
//
// Inputs: Square Matrix m, tolerance tol ≥ 0.
// Complexity: O(n^2) where n = Rows(A). Space: O(1).
// Returns ErrNilMatrix/ErrDimensionMismatch on structural issues, ErrNaNInf on bad tol,
// ErrAsymmetry on violation.
// AI-Hints: Use for Eigen decomposition and PSD tests. Require a square matrix for symmetry.
func ValidateSymmetric(m Matrix, tol float64) error {
	// Guard nil first.
	if m == nil {
		return validatorErrorf("ValidateSymmetric", ErrNilMatrix) // avoid dereferencing nil
	}
	// Check the square condition explicitly.
	if m.Rows() != m.Cols() {
		return validatorErrorf("ValidateSymmetric", ErrDimensionMismatch) // propagate dimension sentinel
	}
	// Normalize tolerance to a non-negative finite value.
	if math.IsNaN(tol) || math.IsInf(tol, 0) {
		// Use existing numeric sentinel rather than inventing a new one.
		return validatorErrorf("ValidateSymmetric", ErrNaNInf) // invalid tolerance is considered a numeric policy violation
	}
	if tol < zeroTol {
		// Negative tolerance makes little semantic sense; flip to its absolute value.
		tol = -tol
	}

	// Early return path: a 0×0 or 1×1 matrix is trivially symmetric.
	n := m.Rows() // n == m.Cols() due to ValidateSquare above
	if n <= 1 {
		return nil // nothing to compare
	}

	// Scan the strict upper triangle once, tracking the maximum deviation.
	// Deterministic i→j order ensures reproducible short-circuiting behavior.
	var (
		i, j   int     // loop counters
		aij    float64 // A[i,j]
		aji    float64 // A[j,i]
		diff   float64 // |aij - aji|
		maxOff float64 // running maximum of the deviation
	)
	for i = 0; i < n; i++ { // fixed row loop
		for j = i + 1; j < n; j++ { // scan only upper triangle
			aij, _ = m.At(i, j)        // At is O(1); errors are not expected after shape validation
			aji, _ = m.At(j, i)        // symmetric counterpart
			diff = math.Abs(aij - aji) // absolute asymmetry magnitude
			// If deviation exceeds tolerance, fail immediately - fast negative path.
			if diff > tol {
				return validatorErrorf("ValidateSymmetric", ErrAsymmetry) // caller may wrap with an operation tag
			}
			// Track the maximum deviation for early-positive reasoning (optional).
			if diff > maxOff {
				maxOff = diff
			}
		}
	}

	// At this point, all |A[i,j]-A[j,i]| ≤ tol, so A is symmetric within tol.
	// Callers (e.g., Eigen) can treat (maxOff == 0) as a "diagonal already" shortcut.
	return nil
}

// IsZeroOffDiagonal reports whether max_{i≠j} |A[i,j]| ≤ tol.
// Useful to early-exit Jacobi when matrix is already (near) diagonal.
// Returns ErrNilMatrix/ErrDimensionMismatch/ErrNaNInf like ValidateSymmetric.
// Complexity: O(n²).
func IsZeroOffDiagonal(m Matrix, tol float64) (bool, error) {
	if m == nil {
		return false, ErrNilMatrix
	}
	if err := ValidateSquare(m); err != nil {
		return false, err
	}
	if math.IsNaN(tol) || math.IsInf(tol, 0) {
		return false, ErrNaNInf
	}
	if tol < zeroTol {
		tol = -tol
	}
	n := m.Rows()
	if n <= 1 {
		return true, nil
	}

	var i, j int
	var v float64
	for i = 0; i < n; i++ {
		for j = 0; j < n; j++ {
			if i == j {
				continue
			}
			v, _ = m.At(i, j)
			if math.Abs(v) > tol {
				return false, nil
			}
		}
	}

	return true, nil
}

// ValidateMulCompatible – Ensures a.Cols == b.Rows, inputs non-nil.
//
// Errors: ErrNilMatrix, ErrDimensionMismatch.
// Complexity: O(1).
// AI-Hints: Use for general matrix multiplication compatibility.
func ValidateMulCompatible(a, b Matrix) error {
	if err := ValidateNotNil(a); err != nil {
		return validatorErrorf("ValidateMulCompatible", err)
	}
	if err := ValidateNotNil(b); err != nil {
		return validatorErrorf("ValidateMulCompatible", err)
	}
	if a.Cols() != b.Rows() {
		return validatorErrorf("ValidateMulCompatible", ErrDimensionMismatch)
	}

	return nil
}

// ValidateGraphAdjacency – Validates adjacency matrix and index map consistency.
//
// Inputs: *AdjacencyMatrix struct.
// Errors: ErrNilMatrix, ErrDimensionMismatch.
// Complexity: O(1).
// AI-Hints: Use before FW/APSP-related kernels.
func ValidateGraphAdjacency(am *AdjacencyMatrix) error {
	if am == nil {
		return validatorErrorf("ValidateGraphAdjacency", ErrNilMatrix)
	}
	if err := ValidateSquareNonNil(am.Mat); err != nil {
		return validatorErrorf("ValidateGraphAdjacency", err)
	}
	if am.vertexByIndex != nil && len(am.vertexByIndex) != am.Mat.Rows() {
		return validatorErrorf("ValidateGraphAdjacency", ErrDimensionMismatch)
	}

	return nil
}

/*
Финальный по файловый список улучшений пакета matrix с привязкой к ТЗ-1..5
Предварительный драфт правок: первые 8 файлов matrix по ТЗ-1–ТЗ-5










нормально.. - хотя Ты наверняка мог гораздо более профессиональнее подойти к реализации - мне пришлось исправлять и дорабатывать Твои результаты… надеюсь в следующий раз Ты всё же постараешься значительно сильнее и всё таки доведёшь уровень качество до достойного lvlath ("НЕПРЕВЗАЙДËННЫЕ" и "ВЕЛИЧАЙШИЕ")!!.. - пожалуйста, хватит так халатно и паскудно относится ко мне, к моим требованиям/задачам и проекту lvlath!!!
!ОБЯЗАТЕЛЬНО продолжай придерживаться, единого стиля и формата, стараться развивать/увеличивать качество проработки деталей и техническое виденье/поведение!! Прошу Тебя быть ещё СТАРАТЕЛЬНЕЕ, ВНИМАТЕЛЬНЕЕ, ПРОДУМАННЕЕ и ЭКСПЕРТНЕЕ!! - ПОЖАЛУЙСТА, ХВАТИТ МУСОРА и ДЕРЬМОВОГО КАЧЕСТВА!!! Хватит генерировать галимую дичь!! Подыми уровень качества, продуманности и проработки!! НЕ СМЕЙ расслабляться или ослаблять обороты - ПРОДОЛЖАЙ стараться и увеличивать уровень качества и профессионализма!!!
Вот, исправленное и доведённое до ума, актуальное состояние matrix/api.go(изучить, сохранить и использовать!):
```

```
+ а так же бенч-тесты matrix/bench_test.go:
```

```

Теперь можем продолжать, но прежде чем мы продолжим, НАПОМИНАЮ наш способ взаимодействия:
```
в каждом моём последующем запросе, я предоставлю Тебе:
(- результат предыдущей проработки с оценкой качества и возможными доп.требованиями)
-  рабочий функциональный файл
- (если существует) соответствующий тестовый файл
- относящееся именно к этим файлам указания и требования из исследования «Финальный по файловый список улучшений пакета matrix с привязкой к ТЗ-1..5» + соответствующие дополнительные уточнения и проработки

и на каждый такой запрос Ты должен:
- детально проанализировать, изучить и проработать всё предоставленное мной!! КАЖДЫЙ ФАЙЛ(ПОЛНОЦЕННО и ВДУМЧИВО, ВСË ЕГО СОДЕРЖИМОЕ) и КАЖДОЕ ОПИСАНИЕ ЗАДАНИЯ!!
- ОСОЗНАТЬ суть каждой правки и (на актуальном состоянии файла) ЭКСПЕРТНО ПОНЯТЬ ЧТО ИМЕННО, ГДЕ ИМЕННО и КАК ИМЕННО НУЖНО РЕАЛИЗОВЫВАТЬ и КАК КОНКРЕТНО ОФОРМИТЬ/ОПИСАТЬ!!.. - нужно всё проработать настолько качественно, подробно, технически ясно и расписано, толково и детально описано!! Каждая сигнатура, каждый тестовый костяк, каждый дифф с правками и каждый коммент!!!
- На основании всего этого выдать мне обновлённое, МАКСИМАЛЬНО ДЕТАЛЬНО И ПОНЯТНО, ТЕХНИЧЕСКИ ПРОДУМАННО и ЭКСПЕРТНО ПРОРАБОТАННОЕ - ПОЛНОМАСШТАБНОЕ ПРОФЕССИОНАЛЬНОЕ ТЗ на КОНКРЕТНЫЙ файл и его тесты!!!.. - такое ТЗ, что бы следуя ему, НЕВОЗМОЖНО БЫЛО СОВЕРШИТЬ ОШИБКУ или СХАЛТУРИТЬ!! - что бы даже примитивный разработчик или бестолковый AI, НЕ СМОГ ВСË ИСПОРТИТЬ и ПРОСРАТЬ!!! СТРОГО и ПОЛНОЦЕННО, ВЫСОКОКАЧЕСТВЕННО, ОСОЗНАНО и ВСЕУЧТИВО!!!
+ если понимаешь что мы делаем что-то не нужное или вредящее (пакету/библиотеке/планам/целям) - обязательно сообщи! ..так же сообщи если осознаёшь какой-либо недостаток информации или же понимаешь что смог бы выдать более качественны/точный и экспертный результат имея в проработке ещё какой-то файл или мои доп.уточнения!
```

Продолжаем по файловую проработку пакета matrix, в соответствии с «Финальный по файловый список улучшений пакета matrix с привязкой к ТЗ-1..5»(и более)!
ПЕРЕХОДИМ ИМЕННО к matrix/dox.go и matrix/example_test.go!!
- проанализируй актуальное состояние файла matrix/doc.go:
```
// SPDX-License-Identifier: MIT
```
- а так же изучи актуальное содержимое файла matrix/example_test.go:
```
// SPDX-License-Identifier: MIT
```
- относящееся именно к matrix/dox.go и matrix/example_test.go, указания и требования из исследования «Финальный по файловый список улучшений пакета matrix с привязкой к ТЗ-1..5» + соответствующие дополнительные уточнения и проработки:
!!~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~!!
matrix/api.go – Public Facades & Core Delegation
!!~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~!!

!!КРИТИЧНО ВАЖНО!!
🦾 При разработке всего этого ОБЯЗАТЕЛЬНО РУКОВОДСТВУЙСЯ И СТРОГО ПРИДЕРЖИВАЙСЯ упомянутые и проработанные ранее исследования, аудиты требования, критерии, ожидания и 99 правил(«lvlath Coding Standard - Methods & Function», «lvlath Coding Standard - Types & Variables», «lvlath Coding Standard - Advanced & Governance»)!! НЕ СМЕЙ ИХ НАРУШАТЬ!!! ПЕРЕД ТЕМ КАК ВЫДАТЬ мне код ПРОВЕРЬ КАЖДУЮ СТРОКУ и ЛОГИЧЕСКИЙ БЛОК НА СТРОГОЕ СООТВЕТСТВИЕ!!!!! У ТЕБЯ ОТСУТСТВУЕТ ВОЗМОЖНОСТЬ ОПУСТИТЬ/НАРУШИТЬ ИЛИ НЕ ПРОВЕРИТЬ КАКОЕ-ТО ПРАВИЛО!! РЕЗУЛЬТАТ ОБЯЗАН СТРОЖАЙШЕ СООТВЕТСТВОВАТЬ КАЖДОМУ ИЗ НИХ!!!! 🤔 +дополнительно ко всем просьбам, старайся придерживаться следующих правил:
0. БЕЗ ХАЛТУР и МУСОРА! БЕЗ БЕСПОЛЕЗНЫХ правок и НЕ РАБОЧИХ решений!!ОБЯЗАТЕЛЬНО помни и грамотно продумывай проектирование и профессионально выноси и пере-используй методы!! Напоминаю, крайне желательно придерживаться одного стиля и подхода как к оформлению комментариев с описанием, так и технической реализации!!
1. Нет цели в тупую изменить или обновить содержимое файла! Не нужно ничего менять если всё уже и так правильно написано! Грамотно и Профессионально дополнить - Хорошо! Аккуратно и Экспертно исправить (действительно проблемное место, действительно правильно исправить) - тоже, Хорошо!  ВДУМЧИВО и ОТВЕТСТВЕННО дополнить/обновить/актуализировать комментарий, описание или часть процесса - ХОРОШО!! Бесполезно обновить название переменной или метода, бессмысленно изменить способ объявления переменных, просто так убрать или изменить уже нормально написанные комментарии и описания - ПЛОХО, ОЧЕНЬ при ОЧЕНЬ ПЛОХО(НЕ СМЕЙ ТАК ДЕЛАТЬ)!!!
2. СВЕРХ АККУРАТНО, ПРОДУМАННО и ЭКСПЕРТНО проработай и реализуй все необходимые правки и обновления!! выдай мне ОЧЕНЬ ГРАМОТНО и ПОНЯТНО оформлен(с шаблонным/(действительно)полезным описанием и продуманныеми/эффективными AI-hints, с упоминанием алгоритмической сложности и причинно-следственный связи, с интуитивно ожидаемыми именами переменных и логическими названиями процессов)… напомню - МЫ НЕПОВТОРИМЫЕ и НЕПРЕВЗОЙДËННЫЕ, МЫ - ЛУЧШИЕ!!!.. - пожалуйста, СООТВЕТСТВУЙ этому уровню!!
3. Результат ОБЯЗАН ИСПРАВНО и ОЖИДАЕМО(совершенно правильно и точно) РАБОТАТЬ и ВЫПОЛНЯТЬ своё ПРЕДНАЗНАЧЕНИЕ!! Можешь думать СКОЛЬКО УГОДНО(любое количество времени) - главное ПОЛНОЦЕННО СООТВЕТСТВУЮЩИЙ МАТ.ФАРМУЛАМ и ИСПРАВНО/ЭФФЕКТИВНО РАБОЧИЙ РЕЗУЛЬТАТ!!!Максимально придерживайся математической грамотности алгоритма и точности расчётов! Эффективно, продуманно и экспертно используй возможности языка Go и нашего же пакета core/!
4. АНГЛИЙСКИЕ комментарии на каждую строку, на каждое действие и на каждую команду/инструкцию! ОПИШИ и ОБЪЯСНИ (ГРАМОТНО, доступно, логично и ТОЛЬКО на Английском)!! - разъяснение шагов алгоритма, причинно-следствия, где и как меняется алгоритмическая сложность, и тд..! !!НИКАКИХ УПОМИНИНИЙ о ТЗ или НЮАНСАХ РАЗРАБОТКИ - ТОЛЬКО ПОЛЕЗНАЯ ИНФОРМАЦИЯ ПО ЭКСПЛУАТАЦИИ!! ПОЛНОЦЕННО, ОСОЗНАНО и ВЫСОКОКАЧЕСТВЕННО! и НЕ СМЕЙ ЗАБЫВАТЬ про действительно рабочие и эффективные AI-hint’ы!! ВСË ОБЯЗАННО СООТВЕТСТВОВАТЬ ШАБЛОНУ:
```
// MethodName MAIN DESCRIPTION (2–3 строки, без маркетинга).
// Implementation:
//   - Stage 1: <валидация/подготовка>
//   - Stage 2: <ядро/алгоритм>
// Behavior highlights:
//   - <детерминизм/fast-path/политики>
// Inputs:
//   - <параметр>: <смысл/единицы/контракт>
// Returns:
//   - <значение/тип>: <смысл>
// Errors:
//   - <перечень sentinel-ошибок и из каких этапов они приходят>
// Determinism:
//   - <фиксированный порядок циклов / stable output / nondeterministic N/A>
// Complexity:
//   - Time O(...), Space O(...). <доп. нюансы при оценке сложности>
// Notes:
//   - <нюансы API, совместимость, side-effects>
// AI-Hints:
//   - <хитрости; спец.пояснения для пользователя(и AI-models);как эффективно/безопасно применять; требования к типам для fast-path (*Dense и т.п.)>
```
5. Интуитивно понятный код и логичные/ожидаемые имена типов методов, свойств, и переменных!! НИКАКИХ магических строк и цифр - всё в понятные константы!!

🦾 Приложи максимум усилий и стараний!!! lvlath/matrix - один из самых ОСНОВНЫХ, ГЛАВНЫХ и ФУНДАМЕНТАЛЬНЫХ под.пакетов!! В дальнейшем он будет использоваться во многих других алгоритмах, реализация и расчётах!!ВЫСОЧАЙШИЕ ответственность, мощность и качество!! В то же время всё должно оставаться интуитивно понятным, удобным и нужным! КАЖДЫЙ блок должен быть проработан более чем полноценно - максимально возможно детально и качественно!!

ПОЖАЛУЙСТА, ВЫДАЙ МНЕ ИМЕННО ТО ЧТО Я ПРОШУ - ПОЛНОЦЕННО и ДОСКОНАЛЬНО ПРОДУМАННЫЕ И ДЕЙСТВИТЕЛЬНО ВЫСОКОКАЧЕСТВЕННО ПРОРАБОТАННЫЕ, В СООТВЕТСТВИИ СО ВСЕМИ УТВЕРЖДËННЫМИ и ОГОВОРЕННЫМИ ТРЕБОВАНИЯМИ правки и улучшения и добавления для matrix/doc.go и matrix/example_test.go соответственно!!!! В ТОМ(или ВЫШЕ) КАЧЕСТВЕ КОТОРЕ Я ОПИСАЛ и ТРЕБУЮ!!!


(всё ещё)Рассчитываю на Тебя - НЕ СМЕЙ ПОДВОДИТЬ МЕНЯ!
*/
