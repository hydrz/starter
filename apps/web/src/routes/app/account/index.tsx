import { createFileRoute } from "@tanstack/react-router";
import { AccountOverviewPage } from "../../../features/account/AccountOverviewPage";

export const Route = createFileRoute("/app/account/")({
  component: AccountOverviewPage,
});
