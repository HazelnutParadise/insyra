import json
import math
import sys
from itertools import permutations

import numpy as np
import scipy.stats as st
import statsmodels.api as sm


def ci_by_alt(center, margin, alt):
    if alt == "greater":
        return [center - margin, float("inf")]
    if alt == "less":
        return [float("-inf"), center + margin]
    return [center - margin, center + margin]


def fisher_ci(r, n, cl=0.95):
    z = 0.5 * math.log((1 + r) / (1 - r))
    se = 1.0 / math.sqrt(n - 3)
    zcrit = st.norm.ppf(1 - (1 - cl) / 2)
    zl = z - zcrit * se
    zu = z + zcrit * se
    el = math.exp(2 * zl)
    eu = math.exp(2 * zu)
    return [(el - 1) / (el + 1), (eu - 1) / (eu + 1)]


def correlation_inference(corr, n):
    t = corr * math.sqrt(n - 2) / math.sqrt(1 - corr * corr)
    p = 2 * (1 - st.t.cdf(abs(t), n - 2))
    ci = fisher_ci(corr, n, 0.95)
    return t, p, float(n - 2), ci


def rank_average(arr):
    return st.rankdata(arr, method="average")


def one_way_stats(groups):
    all_values = np.concatenate([np.array(g, dtype=float) for g in groups])
    labels = []
    for idx, g in enumerate(groups):
        labels.extend([idx] * len(g))
    labels = np.array(labels)

    total_mean = np.mean(all_values)
    ssb = 0.0
    ssw = 0.0
    for i in range(len(groups)):
        vals = all_values[labels == i]
        mean_i = np.mean(vals)
        ssb += len(vals) * (mean_i - total_mean) ** 2
        ssw += np.sum((vals - mean_i) ** 2)

    dfb = len(groups) - 1
    dfw = len(all_values) - len(groups)
    f = (ssb / dfb) / (ssw / dfw)
    p = 1 - st.f.cdf(f, dfb, dfw)
    eta = ssb / (ssb + ssw)
    return ssb, ssw, dfb, dfw, f, p, eta


def two_way_stats(a_levels, b_levels, cells):
    all_values = []
    fa = []
    fb = []
    counts = []
    for i in range(a_levels):
        for j in range(b_levels):
            cell = [float(v) for v in cells[i * b_levels + j]]
            counts.append(len(cell))
            for v in cell:
                all_values.append(v)
                fa.append(i)
                fb.append(j)

    all_values = np.array(all_values, dtype=float)
    fa = np.array(fa)
    fb = np.array(fb)
    total_mean = np.mean(all_values)

    ssa = 0.0
    for i in range(a_levels):
        vals = all_values[fa == i]
        ssa += len(vals) * (np.mean(vals) - total_mean) ** 2

    ssb = 0.0
    for j in range(b_levels):
        vals = all_values[fb == j]
        ssb += len(vals) * (np.mean(vals) - total_mean) ** 2

    cell_means = []
    for i in range(a_levels):
        for j in range(b_levels):
            vals = all_values[(fa == i) & (fb == j)]
            cell_means.append(float(np.mean(vals)))

    ssab = 0.0
    for i in range(a_levels):
        mean_a = np.mean(all_values[fa == i])
        for j in range(b_levels):
            mean_b = np.mean(all_values[fb == j])
            idx = i * b_levels + j
            n_ij = counts[idx]
            mu_ij = cell_means[idx]
            ssab += n_ij * (mu_ij - mean_a - mean_b + total_mean) ** 2

    ssw = 0.0
    for i in range(a_levels):
        for j in range(b_levels):
            idx = i * b_levels + j
            vals = all_values[(fa == i) & (fb == j)]
            ssw += np.sum((vals - cell_means[idx]) ** 2)

    dfa = a_levels - 1
    dfb = b_levels - 1
    dfab = dfa * dfb
    dfw = len(all_values) - (a_levels * b_levels)

    msa = ssa / dfa
    msb = ssb / dfb
    msab = ssab / dfab
    msw = ssw / dfw

    fa_stat = msa / msw
    fb_stat = msb / msw
    fab_stat = msab / msw

    pa = 1 - st.f.cdf(fa_stat, dfa, dfw)
    pb = 1 - st.f.cdf(fb_stat, dfb, dfw)
    pab = 1 - st.f.cdf(fab_stat, dfab, dfw)

    return {
        "ssa": ssa, "ssb": ssb, "ssab": ssab, "ssw": ssw,
        "dfa": float(dfa), "dfb": float(dfb), "dfab": float(dfab), "dfw": float(dfw),
        "fa": fa_stat, "fb": fb_stat, "fab": fab_stat,
        "pa": pa, "pb": pb, "pab": pab,
        "etaa": ssa / (ssa + ssw),
        "etab": ssb / (ssb + ssw),
        "etaab": ssab / (ssab + ssw),
        "total_ss": ssa + ssb + ssab + ssw,
    }


def rm_stats(subjects):
    m = np.array(subjects, dtype=float)
    n = m.shape[0]
    k = m.shape[1]
    grand = np.mean(m)
    cond_means = np.mean(m, axis=0)
    subj_means = np.mean(m, axis=1)

    ss_factor = n * np.sum((cond_means - grand) ** 2)
    ss_subject = k * np.sum((subj_means - grand) ** 2)
    ss_total = np.sum((m - grand) ** 2)
    ss_within = ss_total - ss_factor - ss_subject

    df_factor = k - 1
    df_subject = n - 1
    df_within = df_factor * df_subject

    ms_factor = ss_factor / df_factor
    ms_within = ss_within / df_within
    f = ms_factor / ms_within
    p = 1 - st.f.cdf(f, df_factor, df_within)
    eta = ss_factor / ss_total
    return {
        "ss_factor": ss_factor,
        "ss_subject": ss_subject,
        "ss_within": ss_within,
        "ss_total": ss_total,
        "df_factor": float(df_factor),
        "df_subject": float(df_subject),
        "df_within": float(df_within),
        "f": f,
        "p": p,
        "eta": eta,
    }


def linear_common(y, x_cols):
    yv = np.array(y, dtype=float)
    x = np.column_stack([np.array(col, dtype=float) for col in x_cols])
    x = sm.add_constant(x, has_constant="add")
    model = sm.OLS(yv, x).fit()

    residuals = (yv - model.predict(x)).tolist()
    coeffs = model.params.tolist()
    ses = model.bse.tolist()
    tv = model.tvalues.tolist()
    pv = model.pvalues.tolist()
    ci = model.conf_int(alpha=0.05).tolist()

    out = {
        "coefficients": coeffs,
        "standard_errors": ses,
        "t_values": tv,
        "p_values": pv,
        "confidence_intervals": ci,
        "residuals": residuals,
        "r_squared": float(model.rsquared),
        "adj_r_squared": float(model.rsquared_adj),
    }
    if len(x_cols) == 1:
        out.update({
            "intercept": coeffs[0],
            "slope": coeffs[1],
            "se_intercept": ses[0],
            "se_slope": ses[1],
            "t_intercept": tv[0],
            "t_slope": tv[1],
            "p_intercept": pv[0],
            "p_slope": pv[1],
            "ci_intercept": ci[0],
            "ci_slope": ci[1],
        })
    return out


def poly_reg(y, x, degree):
    xv = np.array(x, dtype=float)
    cols = [np.ones_like(xv)]
    for d in range(1, degree + 1):
        cols.append(np.power(xv, d))
    xmat = np.column_stack(cols)
    yv = np.array(y, dtype=float)
    model = sm.OLS(yv, xmat).fit()
    return {
        "coefficients": model.params.tolist(),
        "standard_errors": model.bse.tolist(),
        "t_values": model.tvalues.tolist(),
        "p_values": model.pvalues.tolist(),
        "confidence_intervals": model.conf_int(alpha=0.05).tolist(),
        "residuals": (yv - model.predict(xmat)).tolist(),
        "r_squared": float(model.rsquared),
        "adj_r_squared": float(model.rsquared_adj),
    }


def exp_reg(y, x):
    xv = np.array(x, dtype=float)
    yv = np.array(y, dtype=float)
    lny = np.log(yv)
    xmat = sm.add_constant(xv, has_constant="add")
    fit = sm.OLS(lny, xmat).fit()
    ln_a = fit.params[0]
    b = fit.params[1]
    a = math.exp(ln_a)

    pred = a * np.exp(b * xv)
    residuals = (yv - pred).tolist()
    sse = float(np.sum((yv - pred) ** 2))
    sst = float(np.sum((yv - np.mean(yv)) ** 2))
    r2 = 1.0 - sse / sst
    n = len(xv)
    df = n - 2
    adj = 1.0 - (1.0 - r2) * ((n - 1) / df)

    mse_log = float(np.sum((lny - fit.predict(xmat)) ** 2) / df)
    mean_x = float(np.mean(xv))
    sxx = float(np.sum((xv - mean_x) ** 2))
    se_b = math.sqrt(mse_log / sxx)
    se_ln_a = math.sqrt(mse_log * (1.0 / n + mean_x * mean_x / sxx))
    se_a = a * se_ln_a

    t_a = a / se_a
    t_b = b / se_b
    p_a = 2 * (1 - st.t.cdf(abs(t_a), df))
    p_b = 2 * (1 - st.t.cdf(abs(t_b), df))
    tcrit = st.t.ppf(0.975, df)
    ci_a = [a - tcrit * se_a, a + tcrit * se_a]
    ci_b = [b - tcrit * se_b, b + tcrit * se_b]

    return {
        "intercept": a,
        "slope": b,
        "residuals": residuals,
        "r_squared": r2,
        "adj_r_squared": adj,
        "se_intercept": se_a,
        "se_slope": se_b,
        "t_intercept": t_a,
        "t_slope": t_b,
        "p_intercept": p_a,
        "p_slope": p_b,
        "ci_intercept": ci_a,
        "ci_slope": ci_b,
    }


def log_reg(y, x):
    xv = np.array(x, dtype=float)
    yv = np.array(y, dtype=float)
    lx = np.log(xv)
    xmat = sm.add_constant(lx, has_constant="add")
    fit = sm.OLS(yv, xmat).fit()
    a = float(fit.params[0])
    b = float(fit.params[1])

    pred = fit.predict(xmat)
    residuals = (yv - pred).tolist()
    sse = float(np.sum((yv - pred) ** 2))
    sst = float(np.sum((yv - np.mean(yv)) ** 2))
    r2 = 1.0 - sse / sst
    n = len(xv)
    df = n - 2
    adj = 1.0 - (1.0 - r2) * ((n - 1) / df)

    mse = sse / df
    mean_lx = float(np.mean(lx))
    sxx = float(np.sum((lx - mean_lx) ** 2))
    se_b = math.sqrt(mse / sxx)
    se_a = math.sqrt(mse * (1.0 / n + mean_lx * mean_lx / sxx))
    t_a = a / se_a
    t_b = b / se_b
    p_a = 2 * (1 - st.t.cdf(abs(t_a), df))
    p_b = 2 * (1 - st.t.cdf(abs(t_b), df))
    tcrit = st.t.ppf(0.975, df)
    ci_a = [a - tcrit * se_a, a + tcrit * se_a]
    ci_b = [b - tcrit * se_b, b + tcrit * se_b]

    return {
        "intercept": a,
        "slope": b,
        "residuals": residuals,
        "r_squared": r2,
        "adj_r_squared": adj,
        "se_intercept": se_a,
        "se_slope": se_b,
        "t_intercept": t_a,
        "t_slope": t_b,
        "p_intercept": p_a,
        "p_slope": p_b,
        "ci_intercept": ci_a,
        "ci_slope": ci_b,
    }


def pca_stats(rows, n_components=None):
    m = np.array(rows, dtype=float)
    n, p = m.shape
    if n_components is None:
        n_components = p
    z = np.copy(m)
    for j in range(p):
        col = z[:, j]
        mean = np.mean(col)
        std = np.std(col, ddof=1)
        if std == 0:
            std = 1.0
        z[:, j] = (col - mean) / std

    cov = np.cov(z, rowvar=False, ddof=1)
    eigvals, eigvecs = np.linalg.eigh(cov)
    idx = np.argsort(eigvals)[::-1]
    eigvals = eigvals[idx]
    eigvecs = eigvecs[:, idx]
    eigvals = eigvals[:n_components]
    eigvecs = eigvecs[:, :n_components]
    for j in range(eigvecs.shape[1]):
        if eigvecs[0, j] < 0:
            eigvecs[:, j] *= -1.0
    explained = (eigvals / np.sum(np.linalg.eigvalsh(cov)) * 100.0).tolist()
    components = eigvecs.T.tolist()
    return {
        "eigenvalues": eigvals.tolist(),
        "explained": explained,
        "components": components,
    }


class RRNG:
    def __init__(self, seed):
        self.mt = [0] * 624
        self.mti = 624
        seed = int(seed) & 0xFFFFFFFF
        for _ in range(50):
            seed = (69069 * seed + 1) & 0xFFFFFFFF
        for i in range(625):
            seed = (69069 * seed + 1) & 0xFFFFFFFF
            if i > 0:
                self.mt[i - 1] = seed

    def mt_genrand(self):
        n = 624
        m = 397
        matrix_a = 0x9908B0DF
        upper_mask = 0x80000000
        lower_mask = 0x7FFFFFFF
        tempering_mask_b = 0x9D2C5680
        tempering_mask_c = 0xEFC60000
        if self.mti >= n:
            mag01 = [0x0, matrix_a]
            for kk in range(0, n - m):
                y = (self.mt[kk] & upper_mask) | (self.mt[kk + 1] & lower_mask)
                self.mt[kk] = self.mt[kk + m] ^ (y >> 1) ^ mag01[y & 0x1]
            for kk in range(n - m, n - 1):
                y = (self.mt[kk] & upper_mask) | (self.mt[kk + 1] & lower_mask)
                self.mt[kk] = self.mt[kk + (m - n)] ^ (y >> 1) ^ mag01[y & 0x1]
            y = (self.mt[n - 1] & upper_mask) | (self.mt[0] & lower_mask)
            self.mt[n - 1] = self.mt[m - 1] ^ (y >> 1) ^ mag01[y & 0x1]
            self.mti = 0
        y = self.mt[self.mti]
        self.mti += 1
        y ^= y >> 11
        y ^= (y << 7) & tempering_mask_b
        y ^= (y << 15) & tempering_mask_c
        y ^= y >> 18
        return float(y) * 2.3283064365386963e-10

    def unif(self):
        i2_32m1 = 2.328306437080797e-10
        x = self.mt_genrand()
        if x <= 0.0:
            return 0.5 * i2_32m1
        if (1.0 - x) <= 0.0:
            return 1.0 - 0.5 * i2_32m1
        return x

    def rbits(self, bits):
        v = 0
        for _ in range(0, bits + 1, 16):
            v1 = int(math.floor(self.unif() * 65536))
            v = 65536 * v + v1
        if bits <= 0:
            return 0.0
        return float(v & ((1 << bits) - 1))

    def unif_index(self, dn):
        if dn <= 0:
            return 0
        bits = int(math.ceil(math.log2(dn)))
        while True:
            dv = self.rbits(bits)
            if dn > dv:
                return int(dv)

    def sample_int(self, n, k):
        pool = list(range(n))
        out = []
        for _ in range(k):
            j = self.unif_index(n)
            out.append(pool[j])
            n -= 1
            pool[j] = pool[n]
        return out


def sq_euclidean(a, b):
    return float(sum((x - y) ** 2 for x, y in zip(a, b)))


def bounded_sq_euclidean(a, b, bound):
    total = 0.0
    for x, y in zip(a, b):
        total += (x - y) ** 2
        if total >= bound:
            return total
    return total


def update_centers_for_transfer(row, centers, nc, an1, an2, from_idx, to_idx):
    al1 = float(nc[from_idx])
    alw = al1 - 1.0
    al2 = float(nc[to_idx])
    alt = al2 + 1.0
    for j in range(len(row)):
        centers[from_idx][j] = (centers[from_idx][j] * al1 - row[j]) / alw
        centers[to_idx][j] = (centers[to_idx][j] * al2 + row[j]) / alt
    nc[from_idx] -= 1
    nc[to_idx] += 1
    an2[from_idx] = alw / al1
    an1[from_idx] = 1.0e30
    if alw > 1.0:
        an1[from_idx] = alw / (alw - 1.0)
    an1[to_idx] = alt / al2
    an2[to_idx] = alt / (alt + 1.0)


def hw_optra(data, centers, ic1, ic2, nc, an1, an2, ncp, d, itran, live, indx):
    n = len(data)
    k = len(centers)
    for l in range(k):
        if itran[l] == 1:
            live[l] = n + 1
    for i, row in enumerate(data):
        indx += 1
        l1 = ic1[i]
        l2 = ic2[i]
        ll = l2
        if nc[l1] == 1:
            continue
        if ncp[l1] != 0:
            d[i] = sq_euclidean(row, centers[l1]) * an1[l1]
        r2 = sq_euclidean(row, centers[l2]) * an2[l2]
        for l in range(k):
            if ((i + 1) >= live[l1] and (i + 1) >= live[l]) or l == l1 or l == ll:
                continue
            rr = r2 / an2[l]
            dc = bounded_sq_euclidean(row, centers[l], rr)
            if dc < rr:
                r2 = dc * an2[l]
                l2 = l
        if r2 >= d[i]:
            ic2[i] = l2
        else:
            indx = 0
            live[l1] = n + i + 1
            live[l2] = n + i + 1
            ncp[l1] = i + 1
            ncp[l2] = i + 1
            update_centers_for_transfer(row, centers, nc, an1, an2, l1, l2)
            ic1[i] = l2
            ic2[i] = l1
        if indx == n:
            return indx
    for l in range(k):
        itran[l] = 0
        live[l] -= n
    return indx


def hw_qtran(data, centers, ic1, ic2, nc, an1, an2, ncp, d, itran, indx_ref, max_qtran):
    n = len(data)
    icoun = 0
    istep = 0
    while True:
        for i, row in enumerate(data):
            icoun += 1
            istep += 1
            if istep >= max_qtran:
                return True
            l1 = ic1[i]
            l2 = ic2[i]
            if nc[l1] == 1:
                if icoun == n:
                    return False
                continue
            if istep <= ncp[l1]:
                d[i] = sq_euclidean(row, centers[l1]) * an1[l1]
            if istep < ncp[l1] or istep < ncp[l2]:
                r2 = d[i] / an2[l2]
                dd = bounded_sq_euclidean(row, centers[l2], r2)
                if dd < r2:
                    icoun = 0
                    indx_ref[0] = 0
                    itran[l1] = 1
                    itran[l2] = 1
                    ncp[l1] = istep + n
                    ncp[l2] = istep + n
                    update_centers_for_transfer(row, centers, nc, an1, an2, l1, l2)
                    ic1[i] = l2
                    ic2[i] = l1
            if icoun == n:
                return False


def build_kmeans_result(data, centers, assignments, nc, iteration, ifault):
    mean = np.mean(np.array(data, dtype=float), axis=0)
    p = len(data[0])
    final_centers = [[0.0] * p for _ in centers]
    for i, row in enumerate(data):
        cluster = assignments[i]
        for j in range(p):
            final_centers[cluster][j] += row[j]
    for l in range(len(final_centers)):
        for j in range(p):
            final_centers[l][j] /= nc[l]
    withinss = [0.0] * len(final_centers)
    totss = 0.0
    for i, row in enumerate(data):
        cluster = assignments[i]
        withinss[cluster] += sq_euclidean(row, final_centers[cluster])
        totss += sq_euclidean(row, mean)
    totwithinss = float(sum(withinss))
    return {
        "cluster": [a + 1 for a in assignments],
        "centers": final_centers,
        "totss": float(totss),
        "withinss": [float(x) for x in withinss],
        "totwithinss": totwithinss,
        "betweenss": float(totss - totwithinss),
        "size": list(nc),
        "iter": int(iteration),
        "ifault": int(ifault),
    }


def single_cluster_result(rows):
    data = [list(map(float, row)) for row in rows]
    center = np.mean(np.array(data, dtype=float), axis=0).tolist()
    totss = 0.0
    for row in data:
        totss += sq_euclidean(row, center)
    return {
        "cluster": [1] * len(data),
        "centers": [center],
        "totss": float(totss),
        "withinss": [float(totss)],
        "totwithinss": float(totss),
        "betweenss": 0.0,
        "size": [len(data)],
        "iter": 1,
        "ifault": 0,
    }


def kmeans_single(rows, init_pool, center_idx, itermax):
    data = [list(map(float, row)) for row in rows]
    k = len(center_idx)
    if k == 1:
        return single_cluster_result(rows)
    centers = [init_pool[i][:] for i in center_idx]
    n = len(data)
    ic1 = [0] * n
    ic2 = [0] * n
    nc = [0] * k
    an1 = [0.0] * k
    an2 = [0.0] * k
    ncp = [0] * k
    d = [0.0] * n
    itran = [0] * k
    live = [0] * k

    for i, row in enumerate(data):
        ic1[i] = 0
        ic2[i] = 1
        dt1 = sq_euclidean(row, centers[0])
        dt2 = sq_euclidean(row, centers[1])
        if dt1 > dt2:
            ic1[i], ic2[i] = 1, 0
            dt1, dt2 = dt2, dt1
        for l in range(2, k):
            db = sq_euclidean(row, centers[l])
            if db >= dt2:
                continue
            if db >= dt1:
                dt2 = db
                ic2[i] = l
                continue
            dt2 = dt1
            ic2[i] = ic1[i]
            dt1 = db
            ic1[i] = l

    p = len(data[0])
    centers = [[0.0] * p for _ in range(k)]
    for i, row in enumerate(data):
        l = ic1[i]
        nc[l] += 1
        for j in range(p):
            centers[l][j] += row[j]
    for l in range(k):
        if nc[l] == 0:
            raise ValueError("empty cluster: try a better set of initial centers")
        aa = float(nc[l])
        for j in range(p):
            centers[l][j] /= aa
        an2[l] = aa / (aa + 1.0)
        an1[l] = 1.0e30
        if aa > 1.0:
            an1[l] = aa / (aa - 1.0)
        itran[l] = 1
        ncp[l] = -1

    ifault = 0
    indx = 0
    max_qtran = 50 * n
    iteration = 0
    for ij in range(1, int(itermax) + 1):
        iteration = ij
        indx = hw_optra(data, centers, ic1, ic2, nc, an1, an2, ncp, d, itran, live, indx)
        if indx == n:
            break
        indx_ref = [indx]
        if hw_qtran(data, centers, ic1, ic2, nc, an1, an2, ncp, d, itran, indx_ref, max_qtran):
            ifault = 4
            break
        indx = indx_ref[0]
        if k == 2:
            break
        for l in range(k):
            ncp[l] = 0
    return build_kmeans_result(data, centers, ic1, nc, iteration, ifault)


def kmeans_stats(rows, k, nstart=1, itermax=10, seed=None):
    data_rows = [list(map(float, row)) for row in rows]
    init_pool = data_rows
    if int(nstart) >= 2:
        deduped = []
        seen = set()
        for row in data_rows:
            key = tuple(row)
            if key in seen:
                continue
            seen.add(key)
            deduped.append(list(key))
        init_pool = deduped
        if len(init_pool) < int(k):
            raise ValueError("more cluster centers than distinct data points")
    rng = RRNG(int(seed if seed is not None else 1))
    best = None
    for _ in range(max(1, int(nstart))):
        center_idx = rng.sample_int(len(init_pool), int(k))
        current = kmeans_single(data_rows, init_pool, center_idx, int(itermax))
        if best is None or current["totwithinss"] < best["totwithinss"]:
            best = current
    return best


def orient_cluster(a, b):
    if abs(a["height"] - b["height"]) > 1e-12:
        return (a, b) if a["height"] < b["height"] else (b, a)
    if a["min_leaf"] < b["min_leaf"]:
        return a, b
    if b["min_leaf"] < a["min_leaf"]:
        return b, a
    return (a, b) if a["rid"] <= b["rid"] else (b, a)


def updated_distance(method, other, a, b, dik, djk, dij):
    if method in ("ward.d", "ward.d2"):
        return (((a["size"] + other["size"]) * dik) + ((b["size"] + other["size"]) * djk) - (other["size"] * dij)) / (a["size"] + b["size"] + other["size"])
    if method == "single":
        return min(dik, djk)
    if method == "complete":
        return max(dik, djk)
    if method == "average":
        return (a["size"] * dik + b["size"] * djk) / (a["size"] + b["size"])
    if method == "mcquitty":
        return 0.5 * dik + 0.5 * djk
    if method == "median":
        return ((dik + djk) - dij / 2.0) / 2.0
    if method == "centroid":
        return (a["size"] * dik + b["size"] * djk - ((a["size"] * b["size"]) * dij) / (a["size"] + b["size"])) / (a["size"] + b["size"])
    return max(dik, djk)


def merged_centroid(a, b, method):
    if method == "median":
        return [0.5 * (x + y) for x, y in zip(a["centroid"], b["centroid"])]
    total = a["size"] + b["size"]
    return [(a["size"] * x + b["size"] * y) / total for x, y in zip(a["centroid"], b["centroid"])]


def tie_break_pair(a1, b1, a2, b2):
    la1, lb1 = orient_cluster(a1, b1)
    la2, lb2 = orient_cluster(a2, b2)
    if la1["min_leaf"] != la2["min_leaf"]:
        return la1["min_leaf"] < la2["min_leaf"]
    return lb1["min_leaf"] < lb2["min_leaf"]


def hclust_stats(rows, method, k=None, h=None):
    data = [list(map(float, row)) for row in rows]
    method = method.lower()
    n = len(data)
    labels = [str(i + 1) for i in range(n)]
    clusters = {}
    active = list(range(n))
    dists = {}
    for i in range(n):
        clusters[i] = {
            "id": i,
            "members": [i],
            "size": 1,
            "centroid": data[i][:],
            "rid": -(i + 1),
            "min_leaf": i,
            "height": 0.0,
        }
    for i in range(n):
        for j in range(i + 1, n):
            dist = float(np.linalg.norm(np.array(data[i]) - np.array(data[j])))
            if method == "ward.d2":
                dist = dist * dist
            dists[(i, j)] = dist
    merge = []
    height = []
    next_id = n
    for step in range(1, n):
        best_i, best_j, best_d = active[0], active[1], float("inf")
        for i in range(len(active)):
            for j in range(i + 1, len(active)):
                a_id, b_id = active[i], active[j]
                key = (a_id, b_id) if a_id < b_id else (b_id, a_id)
                d = dists[key]
                if d < best_d or (abs(d - best_d) <= 1e-12 and tie_break_pair(clusters[a_id], clusters[b_id], clusters[best_i], clusters[best_j])):
                    best_i, best_j, best_d = a_id, b_id, d
        a, b = clusters[best_i], clusters[best_j]
        left, right = orient_cluster(a, b)
        merge.append([left["rid"], right["rid"]])
        height.append(math.sqrt(best_d) if method == "ward.d2" else best_d)
        new_cluster = {
            "id": next_id,
            "members": left["members"] + right["members"],
            "size": left["size"] + right["size"],
            "centroid": merged_centroid(left, right, method),
            "rid": step,
            "min_leaf": min(left["min_leaf"], right["min_leaf"]),
            "height": best_d,
        }
        clusters[next_id] = new_cluster
        new_active = []
        for cid in active:
            if cid in (best_i, best_j):
                continue
            new_active.append(cid)
            key_i = (cid, best_i) if cid < best_i else (best_i, cid)
            key_j = (cid, best_j) if cid < best_j else (best_j, cid)
            dists[(cid, next_id) if cid < next_id else (next_id, cid)] = updated_distance(method, clusters[cid], a, b, dists[key_i], dists[key_j], best_d)
            dists.pop(key_i, None)
            dists.pop(key_j, None)
        dists.pop((best_i, best_j) if best_i < best_j else (best_j, best_i), None)
        new_active.append(next_id)
        active = new_active
        next_id += 1

    root = clusters[active[0]]
    order = [m + 1 for m in root["members"]]
    result = {"merge": merge, "height": height, "order": order, "labels": labels}
    if k is not None:
        result["cut_k"] = cut_tree_from_result(result, k=int(k))
    if h is not None:
        result["cut_h"] = cut_tree_from_result(result, height_cut=float(h))
    return result


def cut_tree_from_result(tree, k=None, height_cut=None):
    n = len(tree["labels"])
    parent = list(range(n))

    def find(x):
        while parent[x] != x:
            parent[x] = parent[parent[x]]
            x = parent[x]
        return x

    def union(a, b):
        ra, rb = find(a), find(b)
        if ra == rb:
            return
        if ra < rb:
            parent[rb] = ra
        else:
            parent[ra] = rb

    nodes = {-(i + 1): [i] for i in range(n)}
    merges_to_apply = n - int(k) if k is not None else None
    for step, row in enumerate(tree["merge"]):
        combined = nodes[row[0]] + nodes[row[1]]
        nodes[step + 1] = combined
        include = (step < merges_to_apply) if merges_to_apply is not None else (tree["height"][step] <= float(height_cut))
        if include and combined:
            base = combined[0]
            for member in combined[1:]:
                union(base, member)

    label_map = {}
    next_label = 1
    out = []
    for i in range(n):
        root = find(i)
        if root not in label_map:
            label_map[root] = next_label
            next_label += 1
        out.append(label_map[root])
    return out


def dbscan_stats(rows, eps, min_pts):
    data = [list(map(float, row)) for row in rows]
    n = len(data)
    neighbors = [[] for _ in range(n)]
    is_seed = [False] * n
    for i in range(n):
        for j in range(n):
            if float(np.linalg.norm(np.array(data[i]) - np.array(data[j]))) <= float(eps):
                neighbors[i].append(j)
        if len(neighbors[i]) >= int(min_pts):
            is_seed[i] = True
    cluster = [0] * n
    visited = [False] * n
    cluster_id = 0
    for i in range(n):
        if visited[i] or not is_seed[i]:
            continue
        cluster_id += 1
        queue = list(neighbors[i])
        cluster[i] = cluster_id
        visited[i] = True
        while queue:
            cur = queue.pop(0)
            if not visited[cur]:
                visited[cur] = True
                if is_seed[cur]:
                    queue.extend(neighbors[cur])
            if cluster[cur] == 0:
                cluster[cur] = cluster_id
    return {"cluster": cluster, "isseed": is_seed}


def silhouette_stats(rows, labels):
    data = [list(map(float, row)) for row in rows]
    labels = [int(x) for x in labels]
    n = len(data)
    dist = np.zeros((n, n), dtype=float)
    for i in range(n):
        for j in range(i):
            d = float(np.linalg.norm(np.array(data[i]) - np.array(data[j])))
            dist[i, j] = d
            dist[j, i] = d
    members = {}
    for i, label in enumerate(labels):
        members.setdefault(label, []).append(i)
    points = []
    total = 0.0
    for i, label in enumerate(labels):
        own = members[label]
        a = 0.0
        if len(own) > 1:
            a = sum(dist[i, j] for j in own if j != i) / (len(own) - 1)
        best_b = float("inf")
        neighbor = 0
        for other_label, idxs in members.items():
            if other_label == label:
                continue
            avg = sum(dist[i, j] for j in idxs) / len(idxs)
            if avg < best_b or (abs(avg - best_b) <= 1e-12 and (neighbor == 0 or other_label < neighbor)):
                best_b = avg
                neighbor = other_label
        s = 0.0
        if len(own) > 1:
            denom = max(a, best_b)
            if denom > 0:
                s = (best_b - a) / denom
        points.append([float(label), float(neighbor), s])
        total += s
    return {"points": points, "avg_width": total / n}


def knn_neighbors_stats(train_rows, test_rows, k):
    train = np.array(train_rows, dtype=float)
    test = np.array(test_rows, dtype=float)
    indices = []
    distances = []
    for row in test:
        d = np.sqrt(np.sum((train - row) ** 2, axis=1))
        order = sorted(range(len(d)), key=lambda i: (float(d[i]), i))
        picked = order[: int(k)]
        indices.append([i + 1 for i in picked])
        distances.append([float(d[i]) for i in picked])
    return {"indices": indices, "distances": distances}


def knn_class_probabilities(indices, distances, labels, classes, weighting):
    probs = [0.0] * len(classes)
    use = list(range(len(indices)))
    if any(abs(distances[i]) <= 1e-12 for i in range(len(distances))):
        use = [i for i in range(len(distances)) if abs(distances[i]) <= 1e-12]
    for pos in use:
        weight = 1.0
        if weighting == "distance" and abs(distances[pos]) > 1e-12:
            weight = 1.0 / distances[pos]
        cls = labels[indices[pos] - 1]
        probs[classes.index(cls)] += weight
    total = sum(probs)
    if total > 0:
        probs = [p / total for p in probs]
    return probs


def knn_classify_stats(train_rows, test_rows, labels, k, weighting):
    labels = [str(v) for v in labels]
    classes = []
    for label in labels:
        if label not in classes:
            classes.append(label)
    nn = knn_neighbors_stats(train_rows, test_rows, k)
    predictions = []
    probabilities = []
    for row_idx in range(len(nn["indices"])):
        idx = nn["indices"][row_idx]
        dist = nn["distances"][row_idx]
        probs = knn_class_probabilities(idx, dist, labels, classes, weighting)
        probabilities.append(probs)
        best = 0
        best_mean = float("inf")
        members = [dist[i] for i in range(len(idx)) if labels[idx[i] - 1] == classes[0]]
        if members:
            best_mean = sum(members) / len(members)
        for c in range(1, len(classes)):
            if probs[c] > probs[best] and abs(probs[c] - probs[best]) > 1e-12:
                best = c
                members = [dist[i] for i in range(len(idx)) if labels[idx[i] - 1] == classes[c]]
                best_mean = sum(members) / len(members) if members else float("inf")
                continue
            if abs(probs[c] - probs[best]) <= 1e-12:
                members = [dist[i] for i in range(len(idx)) if labels[idx[i] - 1] == classes[c]]
                cand_mean = sum(members) / len(members) if members else float("inf")
                if cand_mean < best_mean and abs(cand_mean - best_mean) > 1e-12:
                    best = c
                    best_mean = cand_mean
        predictions.append(classes[best])
    return {"predictions": predictions, "classes": classes, "probabilities": probabilities}


def knn_regress_stats(train_rows, test_rows, targets, k, weighting):
    targets = [float(v) for v in targets]
    nn = knn_neighbors_stats(train_rows, test_rows, k)
    predictions = []
    for row_idx in range(len(nn["indices"])):
        idx = nn["indices"][row_idx]
        dist = nn["distances"][row_idx]
        use = list(range(len(idx)))
        if any(abs(dist[i]) <= 1e-12 for i in range(len(dist))):
            use = [i for i in range(len(dist)) if abs(dist[i]) <= 1e-12]
        weights = []
        values = []
        for pos in use:
            weight = 1.0
            if weighting == "distance" and abs(dist[pos]) > 1e-12:
                weight = 1.0 / dist[pos]
            weights.append(weight)
            values.append(targets[idx[pos] - 1])
        predictions.append(sum(w * v for w, v in zip(weights, values)) / sum(weights))
    return {"predictions": predictions}


def bartlett_sphericity(rows):
    m = np.array(rows, dtype=float)
    n = m.shape[0]
    p = m.shape[1]
    corr = np.corrcoef(m, rowvar=False)
    det = float(np.linalg.det(corr))
    chisq = -((n - 1) - (2 * p + 5) / 6.0) * math.log(det)
    df = (p * (p - 1)) / 2.0
    pval = 1 - st.chi2.cdf(chisq, df)
    return {"chi_square": chisq, "p_value": pval, "df": df}


def corr_pair_stat_p(x, y, method):
    n = len(x)
    if method == "pearson":
        r = float(np.corrcoef(x, y)[0, 1])
        if abs(r) >= 0.9999:
            return r, 0.0
        _, p, _, _ = correlation_inference(r, n)
        return r, p
    if method == "spearman":
        rx = rank_average(x)
        ry = rank_average(y)
        r = float(np.corrcoef(rx, ry)[0, 1])
        if abs(r) >= 0.9999:
            return r, 0.0
        _, p, _, _ = correlation_inference(r, n)
        return r, p
    tau = float(st.kendalltau(x, y, method="asymptotic").statistic)
    if n <= 7:
        obs = abs(tau)
        y_sorted = sorted([float(v) for v in y])
        total = 0
        extreme = 0
        for perm in permutations(y_sorted):
            alt_tau = float(st.kendalltau(x, perm, method="asymptotic").statistic)
            if abs(alt_tau) >= obs:
                extreme += 1
            total += 1
        p = extreme / total
        return tau, p
    z = 3 * tau * math.sqrt(n * (n - 1)) / math.sqrt(2 * (2 * n + 5))
    p = 2 * (1 - st.norm.cdf(abs(z)))
    return tau, p


def corr_matrices(rows, method):
    m = np.array(rows, dtype=float)
    cols = [m[:, j] for j in range(m.shape[1])]
    n = len(cols)
    corr = [[0.0 for _ in range(n)] for _ in range(n)]
    pmat = [[0.0 for _ in range(n)] for _ in range(n)]
    for i in range(n):
        for j in range(i, n):
            if i == j:
                corr[i][j] = 1.0
                pmat[i][j] = 0.0
            else:
                r, p = corr_pair_stat_p(cols[i], cols[j], method)
                corr[i][j] = r
                corr[j][i] = r
                pmat[i][j] = p
                pmat[j][i] = p
    return corr, pmat


def main():
    method = sys.argv[1]
    payload = json.loads(sys.argv[2])

    if method == "single_t":
        x = np.array(payload["x"], dtype=float)
        mu = float(payload["mu"])
        cl = float(payload["cl"])
        n = len(x)
        mean = float(np.mean(x))
        sd = float(np.std(x, ddof=1))
        se = sd / math.sqrt(n)
        t = (mean - mu) / se
        p = 2 * (1 - st.t.cdf(abs(t), n - 1))
        tcrit = st.t.ppf(1 - (1 - cl) / 2, n - 1)
        ci = [mean - tcrit * se, mean + tcrit * se]
        d = (mean - mu) / sd
        out = {"stat": t, "p": p, "df": float(n - 1), "ci": ci, "mean": mean, "effect": d}
    elif method == "two_t":
        x = np.array(payload["x"], dtype=float)
        y = np.array(payload["y"], dtype=float)
        eq = bool(payload["equal_var"])
        cl = float(payload["cl"])
        n1 = len(x)
        n2 = len(y)
        m1 = float(np.mean(x))
        m2 = float(np.mean(y))
        v1 = float(np.var(x, ddof=1))
        v2 = float(np.var(y, ddof=1))
        diff = m1 - m2
        if eq:
            pvar = (((n1 - 1) * v1) + ((n2 - 1) * v2)) / (n1 + n2 - 2)
            se = math.sqrt(pvar * (1 / n1 + 1 / n2))
            df = float(n1 + n2 - 2)
            d = diff / math.sqrt(pvar)
        else:
            se = math.sqrt(v1 / n1 + v2 / n2)
            df = ((v1 / n1 + v2 / n2) ** 2) / (((v1 / n1) ** 2) / (n1 - 1) + ((v2 / n2) ** 2) / (n2 - 1))
            d = diff / math.sqrt((v1 + v2) / 2)
        t = diff / se
        p = 2 * (1 - st.t.cdf(abs(t), df))
        tcrit = st.t.ppf(1 - (1 - cl) / 2, df)
        ci = [diff - tcrit * se, diff + tcrit * se]
        out = {"stat": t, "p": p, "df": df, "ci": ci, "mean1": m1, "mean2": m2, "effect": d}
    elif method == "paired_t":
        x = np.array(payload["x"], dtype=float)
        y = np.array(payload["y"], dtype=float)
        cl = float(payload["cl"])
        dxy = x - y
        n = len(dxy)
        mean_diff = float(np.mean(dxy))
        sd = float(np.std(dxy, ddof=1))
        se = sd / math.sqrt(n)
        t = mean_diff / se
        p = 2 * (1 - st.t.cdf(abs(t), n - 1))
        tcrit = st.t.ppf(1 - (1 - cl) / 2, n - 1)
        ci = [mean_diff - tcrit * se, mean_diff + tcrit * se]
        out = {"stat": t, "p": p, "df": float(n - 1), "ci": ci, "mean_diff": mean_diff, "effect": abs(mean_diff) / sd}
    elif method == "single_z":
        x = np.array(payload["x"], dtype=float)
        mu = float(payload["mu"])
        sigma = float(payload["sigma"])
        alt = payload["alt"]
        cl = float(payload["cl"])
        n = len(x)
        mean = float(np.mean(x))
        se = sigma / math.sqrt(n)
        z = (mean - mu) / se
        if alt == "greater":
            p = 1 - st.norm.cdf(z)
        elif alt == "less":
            p = st.norm.cdf(z)
        else:
            p = 2 * (1 - st.norm.cdf(abs(z)))
        margin = st.norm.ppf(1 - (1 - cl) / 2) * se
        ci = ci_by_alt(mean, margin, alt)
        out = {"stat": z, "p": p, "ci": ci, "mean": mean, "effect": abs(mean - mu) / sigma}
    elif method == "two_z":
        x = np.array(payload["x"], dtype=float)
        y = np.array(payload["y"], dtype=float)
        s1 = float(payload["sigma1"])
        s2 = float(payload["sigma2"])
        alt = payload["alt"]
        cl = float(payload["cl"])
        n1 = len(x)
        n2 = len(y)
        m1 = float(np.mean(x))
        m2 = float(np.mean(y))
        diff = m1 - m2
        se = math.sqrt((s1 * s1) / n1 + (s2 * s2) / n2)
        z = diff / se
        if alt == "greater":
            p = 1 - st.norm.cdf(z)
        elif alt == "less":
            p = st.norm.cdf(z)
        else:
            p = 2 * (1 - st.norm.cdf(abs(z)))
        margin = st.norm.ppf(1 - (1 - cl) / 2) * se
        ci = ci_by_alt(diff, margin, alt)
        pooled_sigma = math.sqrt((n1 * s1 * s1 + n2 * s2 * s2) / (n1 + n2))
        out = {"stat": z, "p": p, "ci": ci, "mean1": m1, "mean2": m2, "effect": abs(diff) / pooled_sigma}
    elif method == "chi_gof":
        vals = [str(v).strip() for v in payload["values"]]
        keys = sorted(list(set(vals)))
        observed = np.array([vals.count(k) for k in keys], dtype=float)
        p_exp = payload["p"]
        if p_exp is None or len(p_exp) == 0:
            p_exp = [1.0 / len(observed)] * len(observed)
        p_exp = np.array(p_exp, dtype=float)
        if bool(payload["rescale"]):
            p_exp = p_exp / np.sum(p_exp)
        expected = np.sum(observed) * p_exp
        chi = float(np.sum((observed - expected) ** 2 / expected))
        df = float(len(observed) - 1)
        pval = 1 - st.chi2.cdf(chi, df)
        out = {
            "stat": chi,
            "p": pval,
            "df": df,
            "observed": observed.tolist(),
            "expected": expected.tolist(),
            "keys": keys,
        }
    elif method == "chi_ind":
        rows = [str(v).strip() for v in payload["rows"]]
        cols = [str(v).strip() for v in payload["cols"]]
        rkeys = sorted(list(set(rows)))
        ckeys = sorted(list(set(cols)))
        obs = np.zeros((len(rkeys), len(ckeys)), dtype=float)
        rmap = {k: i for i, k in enumerate(rkeys)}
        cmap = {k: j for j, k in enumerate(ckeys)}
        for i in range(len(rows)):
            obs[rmap[rows[i]], cmap[cols[i]]] += 1
        rs = np.sum(obs, axis=1).reshape(-1, 1)
        cs = np.sum(obs, axis=0).reshape(1, -1)
        total = np.sum(obs)
        exp = rs @ cs / total
        chi = float(np.sum((obs - exp) ** 2 / exp))
        df = float((len(rkeys) - 1) * (len(ckeys) - 1))
        pval = 1 - st.chi2.cdf(chi, df)
        out = {
            "stat": chi,
            "p": pval,
            "df": df,
            "observed": obs.tolist(),
            "expected": exp.tolist(),
            "row_keys": rkeys,
            "col_keys": ckeys,
        }
    elif method == "oneway_anova":
        groups = payload["groups"]
        ssb, ssw, dfb, dfw, f, p, eta = one_way_stats(groups)
        out = {"ssb": ssb, "ssw": ssw, "dfb": float(dfb), "dfw": float(dfw), "f": f, "p": p, "eta": eta, "total_ss": ssb + ssw}
    elif method == "twoway_anova":
        out = two_way_stats(int(payload["a_levels"]), int(payload["b_levels"]), payload["cells"])
    elif method == "rm_anova":
        out = rm_stats(payload["subjects"])
    elif method == "f_var":
        x = np.array(payload["x"], dtype=float)
        y = np.array(payload["y"], dtype=float)
        v1 = float(np.var(x, ddof=1))
        v2 = float(np.var(y, ddof=1))
        f = v1 / v2 if v1 > v2 else v2 / v1
        df1 = float(len(x) - 1)
        df2 = float(len(y) - 1)
        cdf = st.f.cdf(f, df1, df2)
        p = 2 * min(cdf, 1 - cdf)
        out = {"stat": f, "p": p, "df1": df1, "df2": df2}
    elif method == "levene":
        groups = [np.array(g, dtype=float) for g in payload["groups"]]
        dev_groups = []
        for g in groups:
            med = float(np.median(g))
            dev_groups.append(np.abs(g - med).tolist())
        ssb, ssw, dfb, dfw, f, p, _ = one_way_stats(dev_groups)
        out = {"stat": f, "p": p, "df1": float(dfb), "df2": float(dfw)}
    elif method == "bartlett":
        groups = [np.array(g, dtype=float) for g in payload["groups"]]
        sum_n1 = 0
        pooled_log_var = 0.0
        weight = 0.0
        k = len(groups)
        for g in groups:
            n = len(g)
            v = float(np.var(g, ddof=1))
            if n < 2 or v <= 0:
                continue
            sum_n1 += n - 1
            pooled_log_var += (n - 1) * math.log(v)
            weight += 1.0 / (n - 1)
        mean_var = 0.0
        for g in groups:
            if len(g) >= 2:
                mean_var += (len(g) - 1) * float(np.var(g, ddof=1))
        mean_var /= sum_n1
        t = (sum_n1 * math.log(mean_var)) - pooled_log_var
        corr = 1.0 + (1.0 / (3.0 * (k - 1))) * (weight - 1.0 / sum_n1)
        chi = t / corr
        df = float(k - 1)
        p = 1 - st.chi2.cdf(chi, df)
        out = {"stat": chi, "p": p, "df": df}
    elif method == "f_reg":
        ssr = float(payload["ssr"])
        sse = float(payload["sse"])
        df1 = float(payload["df1"])
        df2 = float(payload["df2"])
        f = (ssr / df1) / (sse / df2)
        p = 1 - st.f.cdf(f, df1, df2)
        out = {"stat": f, "p": p, "df1": df1, "df2": df2}
    elif method == "f_nested":
        rss_r = float(payload["rss_reduced"])
        rss_f = float(payload["rss_full"])
        df_r = int(payload["df_reduced"])
        df_f = int(payload["df_full"])
        ndf = float(df_r - df_f)
        ddf = float(df_f)
        f = ((rss_r - rss_f) / ndf) / (rss_f / ddf)
        p = 1 - st.f.cdf(f, ndf, ddf)
        out = {"stat": f, "p": p, "df1": ndf, "df2": ddf}
    elif method == "covariance":
        x = np.array(payload["x"], dtype=float)
        y = np.array(payload["y"], dtype=float)
        out = {"cov": float(np.cov(x, y, ddof=1)[0, 1])}
    elif method == "correlation":
        x = np.array(payload["x"], dtype=float)
        y = np.array(payload["y"], dtype=float)
        m = payload["corr_method"]
        n = len(x)
        if m == "pearson":
            r = float(np.corrcoef(x, y)[0, 1])
            if abs(r) >= 0.9999:
                out = {"stat": r, "p": 0.0, "df": float(n - 2)}
            else:
                _, p, df, ci = correlation_inference(r, n)
                out = {"stat": r, "p": p, "df": df, "ci": ci}
        elif m == "spearman":
            rx = rank_average(x)
            ry = rank_average(y)
            r = float(np.corrcoef(rx, ry)[0, 1])
            if abs(r) >= 0.9999:
                out = {"stat": r, "p": 0.0}
            else:
                _, p, df, ci = correlation_inference(r, n)
                out = {"stat": r, "p": p, "df": df, "ci": ci}
        else:
            tau, p = corr_pair_stat_p(x, y, "kendall")
            out = {"stat": tau, "p": p}
    elif method == "bartlett_sphericity":
        out = bartlett_sphericity(payload["rows"])
    elif method == "corr_matrix":
        corr, pmat = corr_matrices(payload["rows"], payload["corr_method"])
        out = {"corr_matrix": corr, "p_matrix": pmat}
    elif method == "corr_analysis":
        corr, pmat = corr_matrices(payload["rows"], payload["corr_method"])
        if payload["corr_method"] == "pearson":
            b = bartlett_sphericity(payload["rows"])
            out = {"corr_matrix": corr, "p_matrix": pmat, "chi_square": b["chi_square"], "p_value": b["p_value"], "df": b["df"]}
        else:
            out = {"corr_matrix": corr, "p_matrix": pmat, "chi_square": "NaN", "p_value": "NaN", "df": 0.0}
    elif method == "linear_reg":
        out = linear_common(payload["y"], payload["xs"])
    elif method == "poly_reg":
        out = poly_reg(payload["y"], payload["x"], int(payload["degree"]))
    elif method == "exp_reg":
        out = exp_reg(payload["y"], payload["x"])
    elif method == "log_reg":
        out = log_reg(payload["y"], payload["x"])
    elif method == "pca":
        out = pca_stats(payload["rows"], payload.get("n_components", None))
    elif method == "kmeans":
        out = kmeans_stats(payload["rows"], payload["k"], payload.get("nstart", 1), payload.get("itermax", 10), payload.get("seed", 1))
    elif method == "hclust":
        out = hclust_stats(payload["rows"], payload["method"], payload.get("k", None), payload.get("h", None))
    elif method == "dbscan":
        out = dbscan_stats(payload["rows"], payload["eps"], payload["min_pts"])
    elif method == "silhouette":
        out = silhouette_stats(payload["rows"], payload["labels"])
    elif method == "knn_classify":
        out = knn_classify_stats(payload["train_rows"], payload["test_rows"], payload["labels"], payload["k"], payload["weighting"])
    elif method == "knn_regress":
        out = knn_regress_stats(payload["train_rows"], payload["test_rows"], payload["targets"], payload["k"], payload["weighting"])
    elif method == "knn_neighbors":
        out = knn_neighbors_stats(payload["train_rows"], payload["test_rows"], payload["k"])
    elif method == "moment":
        x = np.array(payload["x"], dtype=float)
        order = int(payload["order"])
        central = bool(payload["central"])
        if central:
            mu = float(np.mean(x))
            out = {"value": float(np.mean((x - mu) ** order))}
        else:
            out = {"value": float(np.mean(x ** order))}
    elif method == "skewness":
        x = np.array(payload["x"], dtype=float)
        mode = payload["mode"]
        n = float(len(x))
        mu = float(np.mean(x))
        m2 = float(np.mean((x - mu) ** 2))
        m3 = float(np.mean((x - mu) ** 3))
        g1 = m3 / (m2 ** 1.5) if m2 != 0 else 0.0
        if mode == "g1":
            out = {"value": g1}
        elif mode == "adjusted":
            out = {"value": g1 * math.sqrt(n * (n - 1)) / (n - 2)}
        else:
            out = {"value": g1 * (((n - 1) / n) ** 1.5)}
    elif method == "kurtosis":
        x = np.array(payload["x"], dtype=float)
        mode = payload["mode"]
        n = float(len(x))
        mu = float(np.mean(x))
        m2 = float(np.mean((x - mu) ** 2))
        m4 = float(np.mean((x - mu) ** 4))
        g2 = m4 / (m2 * m2) - 3.0
        if mode == "g2":
            out = {"value": g2}
        elif mode == "adjusted":
            out = {"value": ((g2 * (n + 1) + 6) * (n - 1)) / ((n - 2) * (n - 3))}
        else:
            out = {"value": (g2 + 3) * ((1 - 1 / n) ** 2) - 3}
    else:
        raise ValueError(f"unsupported method: {method}")

    def sanitize(v):
        if isinstance(v, float):
            if math.isinf(v):
                return "Inf" if v > 0 else "-Inf"
            if math.isnan(v):
                return "NaN"
            return v
        if isinstance(v, list):
            return [sanitize(x) for x in v]
        if isinstance(v, dict):
            return {k: sanitize(val) for k, val in v.items()}
        return v

    print(json.dumps(sanitize(out), allow_nan=False))


if __name__ == "__main__":
    main()
