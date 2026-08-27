import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

/**
 * A destructive confirmation for actions the server does NOT gate behind a typed
 * phrase.
 *
 * The type-the-name control in settings/danger-zone mirrors an endpoint that
 * demands the same phrase in its body; copying it here would invent a gate the
 * API does not honour, and a gate that exists only on screen teaches operators
 * that the real one is somewhere else. What these actions need instead is the
 * consequences stated before the click, because both are quiet: removing an
 * agent leaves the process running on the host and failing authentication, and
 * neither prints anything the operator would otherwise see.
 */
export function ConfirmDestructive({
  open,
  onOpenChange,
  title,
  description,
  consequences,
  confirmLabel,
  pendingLabel,
  pending,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string;
  consequences: string[];
  confirmLabel: string;
  pendingLabel: string;
  pending: boolean;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="text-destructive">{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <ul className="space-y-1 text-sm text-muted-foreground">
          {consequences.map((line) => (
            <li key={line} className="flex gap-2">
              <span aria-hidden="true" className="text-destructive">
                &bull;
              </span>
              <span>{line}</span>
            </li>
          ))}
        </ul>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" disabled={pending} onClick={onConfirm}>
            {pending ? pendingLabel : confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
