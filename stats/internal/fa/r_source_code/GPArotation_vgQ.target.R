# GPArotation::vgQ.target  (namespace: GPArotation)
# dumped at 2025-10-04 13:16:30.711442
vgQ.target <- 
function (L, Target = NULL) 
{
    if (is.null(Target)) 
        stop("argument Target must be specified.")
    Gq <- 2 * (L - Target)
    Gq[is.na(Gq)] <- 0
    list(Gq = Gq, f = sum((L - Target)^2, na.rm = TRUE), Method = "Target rotation")
}

