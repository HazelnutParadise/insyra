# GPArotation::vgQ.quartimax  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.705812
vgQ.quartimax <- 
function (L) 
{
    list(Gq = -L^3, f = -sum(diag(crossprod(L^2)))/4, Method = "Quartimax")
}

