import {ReactNode} from 'react';
import {Button} from './Button';
import './Modal.css';

interface ConfirmModalProps {
    open: boolean;
    title: string;
    description?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    onConfirm: () => void;
    onCancel: () => void;
    children?: ReactNode;
}

export const ConfirmModal = ({
    open,
    title,
    description,
    confirmLabel = 'Bestätigen',
    cancelLabel = 'Abbrechen',
    onConfirm,
    onCancel,
}: ConfirmModalProps) => {
    if (!open) return null;

    return (
        <div className="od-modal-overlay" onClick={onCancel}>
            <div className="od-modal" onClick={(e) => e.stopPropagation()}>
                <h2 className="od-modal-title">{title}</h2>
                {description && <p className="od-modal-desc">{description}</p>}
                <div className="od-modal-actions">
                    <Button variant="outline" size="sm" onClick={onCancel}>
                        {cancelLabel}
                    </Button>
                    <Button variant="primary" size="sm" onClick={onConfirm}>
                        {confirmLabel}
                    </Button>
                </div>
            </div>
        </div>
    );
};
