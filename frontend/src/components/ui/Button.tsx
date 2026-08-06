import {ButtonHTMLAttributes, forwardRef} from 'react';
import './ui.css';

type Variant = 'primary' | 'outline' | 'ghost' | 'danger';
type Size = 'sm' | 'md' | 'lg' | 'icon';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: Variant;
    size?: Size;
}

const variantClass: Record<Variant, string> = {
    primary: 'od-btn-primary',
    outline: 'od-btn-outline',
    ghost: 'od-btn-ghost',
    danger: 'od-btn-danger',
};

const sizeClass: Record<Size, string> = {
    sm: 'od-btn-sm',
    md: 'od-btn-md',
    lg: 'od-btn-lg',
    icon: 'od-btn-icon',
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
    ({variant = 'primary', size = 'md', className, ...props}, ref) => {
        return (
            <button
                ref={ref}
                className={`od-btn ${variantClass[variant]} ${sizeClass[size]} ${className ?? ''}`}
                {...props}
            />
        );
    }
);

Button.displayName = 'Button';
