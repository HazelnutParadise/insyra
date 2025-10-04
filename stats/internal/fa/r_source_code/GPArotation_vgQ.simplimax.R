# GPArotation::vgQ.simplimax  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.709007
vgQ.simplimax <- 
function (L, k = nrow(L)) 
{
    Imat <- sign(L^2 <= sort(L^2)[k])
    list(Gq = 2 * Imat * L, f = sum(Imat * L^2), Method = "Simplimax")
}

