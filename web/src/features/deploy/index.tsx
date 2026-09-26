import type { ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import { CircleAlert, CircleCheck, Download, Info, XCircle } from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useSite } from "@/hooks/useContents";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { saveFile, useExportSite } from "@/hooks/useExport";
import { canPublish, useDelivery, usePublish } from "@/hooks/usePublish";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { DeliveryStages } from "@/components/publish/Delivery";

/**
 * Deploying is how a site written here gets online. A static site has two
 * ways: an archive uploaded by hand, which needs nothing but a place to put
 * files, and a push to GitHub, whose Pages then builds and serves it.
 */
export function Deploy() {
  const { t } = useI18n();
  useDocumentTitle(t("nav.deploy"));

  return (
    <>
      <AppHeader />
      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <PageTitle title={t("nav.deploy")} description={t("deploy.note")} />
        <div className="grid max-w-3xl gap-4 sm:gap-6">
          <ExportCard />
          <GitHubCard />
        </div>
      </Main>
    </>
  );
}

function ExportCard() {
  const { t } = useI18n();
  const problem = useProblem();
  const site = useSite();
  const exporting = useExportSite();

  const address = site.data?.base_url ?? "";
  const unreadable = site.data?.problems?.length ?? 0;

  const start = () =>
    exporting.mutate(undefined, {
      onSuccess: ({ archive, name }) => {
        saveFile(archive, name);
        toast.success(t("deploy.exported", { name, size: sizeOf(archive.size) }));
      },
    });

  const failure = exporting.error;
  const said =
    failure instanceof ApiError
      ? problem(failure.code, failure.message)
      : failure
        ? { title: t("deploy.exportFailed"), detail: String(failure) }
        : null;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("deploy.export")}</CardTitle>
        <CardDescription>{t("deploy.exportNote")}</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-4">
        {site.isPending ? (
          <Skeleton className="h-16 w-full" />
        ) : (
          <ul className="grid gap-2 text-sm">
            {isLocal(address) ? (
              <Check tone="warn">
                {t("deploy.addressLocal", { url: address || "—" })}{" "}
                <Link to="/settings" className="font-medium text-brand hover:underline">
                  {t("deploy.addressFix")}
                </Link>
              </Check>
            ) : (
              <Check tone="ok">{t("deploy.addressOK", { url: address })}</Check>
            )}
            <Check tone="info">{t("deploy.draftsNote")}</Check>
            {unreadable > 0 && <Check tone="bad">{t("deploy.problems", { count: unreadable })}</Check>}
          </ul>
        )}

        {said && (
          <Alert variant="destructive">
            <XCircle />
            <AlertTitle>{said.title}</AlertTitle>
            {said.detail && <AlertDescription className="break-all">{said.detail}</AlertDescription>}
          </Alert>
        )}

        <div>
          <Button onClick={start} disabled={exporting.isPending}>
            {exporting.isPending ? <Spinner /> : <Download />}
            {t(exporting.isPending ? "deploy.exporting" : "deploy.exportAction")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function GitHubCard() {
  const { t } = useI18n();
  const delivery = useDelivery();
  const publish = usePublish([]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("deploy.github")}</CardTitle>
        <CardDescription>{t("deploy.githubNote")}</CardDescription>
      </CardHeader>
      <CardContent className="grid gap-4 text-sm">
        {delivery.isPending ? (
          <Skeleton className="h-24 w-full" />
        ) : canPublish(delivery.data) ? (
          <>
            <p className="text-muted-foreground">{t("deploy.githubReady")}</p>
            <DeliveryStages delivery={delivery.data} publish={publish} />
          </>
        ) : (
          <>
            <p>{t("deploy.githubSteps")}</p>
            <ol className="grid list-decimal gap-3 ps-5 marker:text-muted-foreground">
              <li>{t("deploy.step1")}</li>
              <li>
                {t("deploy.step2")}
                <pre className="mt-2 overflow-x-auto rounded-md bg-muted px-3 py-2 font-mono text-xs leading-relaxed">
                  {[
                    "git init",
                    "git add .",
                    'git commit -m "first commit"',
                    "git branch -M main",
                    "git remote add origin https://github.com/<you>/<repo>.git",
                    "git push -u origin main",
                  ].join("\n")}
                </pre>
              </li>
              <li>{t("deploy.step3")}</li>
              <li>{t("deploy.step4")}</li>
            </ol>
          </>
        )}
      </CardContent>
    </Card>
  );
}

const checkTones = {
  ok: { icon: CircleCheck, className: "text-success" },
  warn: { icon: CircleAlert, className: "text-warning" },
  bad: { icon: XCircle, className: "text-destructive" },
  info: { icon: Info, className: "text-muted-foreground" },
};

/** Check is one thing the export will do, and whether it is as it should be. */
function Check({ tone, children }: { tone: keyof typeof checkTones; children: ReactNode }) {
  const { icon: Icon, className } = checkTones[tone];
  return (
    <li className="flex items-start gap-2">
      <Icon className={cn("mt-0.5 size-4 shrink-0", className)} />
      <span>{children}</span>
    </li>
  );
}

/**
 * isLocal reports an address only this computer can reach, which an exported
 * site would still write into its feeds and sitemap.
 */
function isLocal(address: string): boolean {
  let host: string;
  try {
    host = new URL(address).hostname;
  } catch {
    return true;
  }
  return (
    host === "" ||
    host === "localhost" ||
    host.endsWith(".localhost") ||
    host === "0.0.0.0" ||
    host === "[::1]" ||
    host.startsWith("127.")
  );
}

/** sizeOf writes a byte count the way a download list does. */
function sizeOf(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}
