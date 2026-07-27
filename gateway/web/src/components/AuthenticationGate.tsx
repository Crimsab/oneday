import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { ApiRequestError, bootstrapBrowserSession } from "../api";

interface AuthenticationGateProps {
  checking: boolean;
  bootstrapAvailable: boolean;
}

export function AuthenticationGate({
  checking,
  bootstrapAvailable,
}: AuthenticationGateProps) {
  const { t } = useTranslation("authentication");
  const [token, setToken] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const credential = token.trim();
    if (!credential || submitting) return;
    setSubmitting(true);
    setError("");
    try {
      await bootstrapBrowserSession(credential);
      window.location.reload();
    } catch (caught) {
      if (caught instanceof ApiRequestError && caught.status === 401) {
        setError(t("invalid"));
      } else {
        setError(t("failed"));
      }
      setSubmitting(false);
    }
  };

  return (
    <main className="authentication-shell">
      <section
        className="authentication-panel"
        aria-labelledby="authentication-title"
        aria-live="polite"
      >
        <div className="authentication-brand" aria-hidden="true">OD</div>
        <div className="authentication-copy">
          <p className="authentication-kicker">{t("kicker")}</p>
          <h1 id="authentication-title">
            {checking ? t("checkingTitle") : t("title")}
          </h1>
          <p>
            {checking
              ? t("checkingDescription")
              : bootstrapAvailable
                ? t("description")
                : t("unavailable")}
          </p>
        </div>

        {!checking && bootstrapAvailable && (
          <form className="authentication-form" onSubmit={submit}>
            <label htmlFor="oneday-browser-token">{t("tokenLabel")}</label>
            <input
              id="oneday-browser-token"
              name="oneday-browser-token"
              type="password"
              autoComplete="current-password"
              value={token}
              onChange={(event) => setToken(event.target.value)}
              aria-describedby="oneday-browser-token-hint"
              aria-invalid={Boolean(error)}
              autoFocus
            />
            <p id="oneday-browser-token-hint" className="authentication-hint">
              {t("tokenHint")}
            </p>
            {error && <p className="authentication-error" role="alert">{error}</p>}
            <button className="primary-button" type="submit" disabled={!token.trim() || submitting}>
              {submitting ? t("submitting") : t("submit")}
            </button>
          </form>
        )}

        {!checking && (
          <p className="authentication-footnote">{t("privacy")}</p>
        )}
      </section>
    </main>
  );
}
