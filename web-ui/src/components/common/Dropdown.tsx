import { useState, useRef, useEffect, useCallback, useLayoutEffect } from 'react';
import { createPortal } from 'react-dom';

interface DropdownProps {
  trigger: React.ReactNode;
  children: React.ReactNode;
  align?: 'left' | 'right';
}

function Dropdown({ trigger, children, align = 'right' }: DropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const triggerRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const updatePosition = useCallback(() => {
    if (!triggerRef.current) return;

    const rect = triggerRef.current.getBoundingClientRect();
    const menuWidth = 192; // w-48 = 12rem = 192px
    const menuHeight = 200; // Approximate max height

    let top = rect.bottom + 4;
    let left = align === 'right' ? rect.right - menuWidth : rect.left;

    // Ensure menu stays within viewport horizontally
    if (left < 8) {
      left = 8;
    } else if (left + menuWidth > window.innerWidth - 8) {
      left = window.innerWidth - menuWidth - 8;
    }

    // If menu would go off bottom of screen, show it above the trigger
    if (top + menuHeight > window.innerHeight - 8) {
      top = rect.top - menuHeight - 4;
      if (top < 8) {
        top = 8;
      }
    }

    setPosition({ top, left });
  }, [align]);

  const handleToggle = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsOpen(prev => {
      if (!prev) {
        // Will update position in useLayoutEffect
      }
      return !prev;
    });
  }, []);

  const handleClose = useCallback(() => {
    setIsOpen(false);
  }, []);

  // Update position when opening (useLayoutEffect to avoid flicker)
  useLayoutEffect(() => {
    if (isOpen) {
      updatePosition();
    }
  }, [isOpen, updatePosition]);

  // Close on outside click
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (e: MouseEvent) => {
      if (
        triggerRef.current &&
        !triggerRef.current.contains(e.target as Node) &&
        menuRef.current &&
        !menuRef.current.contains(e.target as Node)
      ) {
        handleClose();
      }
    };

    // Only close on window scroll, not on internal element scrolls
    const handleWindowScroll = () => {
      handleClose();
    };

    document.addEventListener('mousedown', handleClickOutside);
    window.addEventListener('scroll', handleWindowScroll);
    window.addEventListener('resize', handleClose);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      window.removeEventListener('scroll', handleWindowScroll);
      window.removeEventListener('resize', handleClose);
    };
  }, [isOpen, handleClose]);

  // Close on escape
  useEffect(() => {
    if (!isOpen) return;

    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        handleClose();
      }
    };

    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, [isOpen, handleClose]);

  return (
    <div ref={triggerRef} className="inline-block">
      <div onClick={handleToggle}>{trigger}</div>
      {isOpen &&
        createPortal(
          <div
            ref={menuRef}
            className="fixed w-48 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm shadow-lg py-1"
            style={{ top: position.top, left: position.left, zIndex: 9999 }}
            onClick={handleClose}
          >
            {children}
          </div>,
          document.body
        )}
    </div>
  );
}

export default Dropdown;
