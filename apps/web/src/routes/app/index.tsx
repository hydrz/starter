import { createFileRoute } from "@tanstack/react-router";
import { OrgPickerPage } from "../../features/organizations/OrgPickerPage";

export const Route = createFileRoute("/app/")({
  component: OrgPickerPage,
});
