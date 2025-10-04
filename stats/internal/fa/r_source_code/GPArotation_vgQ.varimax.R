# GPArotation::vgQ.varimax  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.712658
vgQ.varimax <- 
function (L) 
{
    QL <- sweep(L^2, 2, colMeans(L^2), "-")
    list(Gq = -L * QL, f = -sqrt(sum(diag(crossprod(QL))))^2/4, 
        Method = "varimax")
}

