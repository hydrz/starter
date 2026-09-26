import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { Code } from "../../components/ui/code";
import * as m from "../../paraglide/messages";
import { OneTimeSecretDialog } from "./OneTimeSecretDialog";

/**
 * 一次性展示状态机的核心不变量（DESIGN.md §8）：明文只在"揭示"那一刻的
 * 那一次渲染里出现，一旦对话框被确认关闭，父组件必须清空它持有的明文，
 * 组件树里就再也找不到这段文本——不是"隐藏起来但还在 DOM/state 里"。
 */
function Harness({ secret }: { secret: string }) {
  const [revealed, setRevealed] = useState<string | null>(secret);
  return (
    <OneTimeSecretDialog
      open={revealed !== null}
      onAcknowledge={() => setRevealed(null)}
      title="One-time secret"
      description="Copy it now."
    >
      <Code copyable={false}>{revealed ?? ""}</Code>
    </OneTimeSecretDialog>
  );
}

describe("OneTimeSecretDialog one-time-reveal state machine", () => {
  it("shows the secret while open, then permanently clears it once acknowledged", async () => {
    const user = userEvent.setup();
    render(<Harness secret="sk_live_abc123" />);

    expect(screen.getByText("sk_live_abc123")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: m.one_time_secret_acknowledge() }),
    );

    expect(screen.queryByText("sk_live_abc123")).not.toBeInTheDocument();
  });

  it("also clears the secret when dismissed via the dialog's own close control (not just the acknowledge button)", async () => {
    const user = userEvent.setup();
    render(<Harness secret="sk_live_xyz789" />);

    expect(screen.getByText("sk_live_xyz789")).toBeInTheDocument();

    // Radix Dialog's built-in close button (rendered by ui/dialog.tsx,
    // labeled "关闭" there), not the "I've saved it" footer button the
    // test above already covers.
    await user.click(screen.getByRole("button", { name: "关闭" }));

    expect(screen.queryByText("sk_live_xyz789")).not.toBeInTheDocument();
  });
});
