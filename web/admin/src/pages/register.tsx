import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { useAuth } from "@/hooks/use-auth";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

export default function RegisterPage() {
  const { register } = useAuth();
  const { t } = useI18n();
  const navigate = useNavigate();
  const [form, setForm] = useState({
    username: "",
    email: "",
    password: "",
    confirmPassword: "",
  });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (form.password !== form.confirmPassword) {
      setError(t("auth.passwordMismatch"));
      return;
    }

    setLoading(true);
    try {
      await register(form.username, form.email, form.password);
      navigate("/login");
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t("auth.registrationFailed");
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  const update = (field: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((prev) => ({ ...prev, [field]: e.target.value }));

  return (
    <Card className="w-full max-w-sm">
      <CardHeader className="text-center">
        <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-lg bg-primary text-primary-foreground font-bold">
          K
        </div>
        <CardTitle className="text-xl">{t("auth.createAccount")}</CardTitle>
        <CardDescription>{t("auth.getStarted")}</CardDescription>
      </CardHeader>
      <form onSubmit={handleSubmit}>
        <CardContent className="space-y-4">
          {error && (
            <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}
          <div className="space-y-2">
            <Label htmlFor="username">{t("auth.username")}</Label>
            <Input
              id="username"
              value={form.username}
              onChange={update("username")}
              placeholder={t("auth.chooseUsername")}
              required
              minLength={3}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="email">{t("auth.email")}</Label>
            <Input
              id="email"
              type="email"
              value={form.email}
              onChange={update("email")}
              placeholder={t("auth.emailPlaceholder")}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">{t("auth.password")}</Label>
            <Input
              id="password"
              type="password"
              value={form.password}
              onChange={update("password")}
              placeholder={t("auth.passwordHint")}
              required
              minLength={6}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="confirmPassword">{t("auth.confirmPassword")}</Label>
            <Input
              id="confirmPassword"
              type="password"
              value={form.confirmPassword}
              onChange={update("confirmPassword")}
              placeholder={t("auth.confirmPasswordPlaceholder")}
              required
            />
          </div>
        </CardContent>
        <CardFooter className="flex flex-col gap-3">
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? t("auth.creatingAccount") : t("auth.register")}
          </Button>
          <p className="text-center text-sm text-muted-foreground">
            {t("auth.hasAccount")}{" "}
            <Link to="/login" className="text-primary hover:underline">
              {t("auth.login")}
            </Link>
          </p>
        </CardFooter>
      </form>
    </Card>
  );
}
