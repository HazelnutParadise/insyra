# Design: cli-test-nan-equality

Two numeric cells are equal when both are `NaN`, or neither is and they differ by at most `tol`. `nil` and non-numeric cells keep their current rules. The helper is shared, ensuring test comparisons never treat `NaN` as equal to a valid number.
