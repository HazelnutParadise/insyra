# GPArotation::vgQ.bentler  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.689916
vgQ.bentler <- 
function (L) 
{
    L2 <- L^2
    M <- crossprod(L2)
    D <- diag(diag(M))
    list(Gq = -L * (L2 %*% (solve(M) - solve(D))), f = -(log(det(M)) - 
        log(det(D)))/4, Method = "Bentler's criterion")
}

