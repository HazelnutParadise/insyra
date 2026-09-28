# Design: cli-test-nan-equality

Two numeric cells are equal when both are `NaN`, or neither is and they differ by at most `tol`. `nil` and non-numeric cells keep their current rules. The helper is shared, so fixing it may turn passing tests red; each such test is read against the code before anything changes, because a wrong `NaN` may be the defect the old helper was hiding.
