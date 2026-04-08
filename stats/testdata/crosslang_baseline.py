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
