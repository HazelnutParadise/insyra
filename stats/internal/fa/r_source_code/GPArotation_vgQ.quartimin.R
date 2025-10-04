# GPArotation::vgQ.quartimin  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.707164
vgQ.quartimin <- 
function (L) 
{
    X <- L^2 %*% (!diag(TRUE, ncol(L)))
    list(Gq = L * X, f = sum(L^2 * X)/4, Method = "Quartimin")
}

