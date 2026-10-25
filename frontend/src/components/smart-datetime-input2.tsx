"use client";

import { useState } from "react";
import {
  formatDateTime,
  parseDateTime,
} from "@/lib/datetime";
import { DateTimePicker24h } from "./datetime-picker-24h";
import { Input } from "./ui/input";
import { ButtonGroup } from "./ui/button-group";

type NaturalLanguageDateTimeInputProps = {
  value: Date | null;
  setValue: (date: Date) => void;
  placeholder?: string;
};

function NaturalLanguageDateTimeInput({ value, setValue, placeholder }: NaturalLanguageDateTimeInputProps) {
  const [text, setText] = useState<string | null>(null)

  const displayValue =
    text !== null
      ? text
      : value
      ? formatDateTime(value)
      : ""
  
  const handleParse = () => {
    if (text === null || !text) return
  
    const parsed = parseDateTime(text)
    if (parsed) {
      setValue(parsed)
      setText(null)
    }
  }

  return (
    <Input
      type="text"
      placeholder={placeholder}
      value={displayValue}
      onChange={(e) => setText(e.target.value)}
      onBlur={() => handleParse()}
      onKeyDown={(e) => {
        if (e.key === "Enter") {
          e.preventDefault()
          handleParse()
        }
      }}
    />
  )
}

type SmartDatetimePickerByTuiProps = {
  value?: Date | null;
  onChange?: (date: Date | null) => void;
  className?: string;
  placeholder?: string;
};

export default function SmartDatetimePickerByTui({
  value,
  onChange,
  className,
  placeholder,
}: SmartDatetimePickerByTuiProps) {
  const [internalValue, setInternalValue] = useState<Date | null>(null);

  const isControlled = value !== undefined;

  const current = isControlled ? value : internalValue;

  const setValue = (date: Date | null) => {
    if (!isControlled) {
      setInternalValue(date);
    }
    onChange?.(date);
  };

  return (
    <div className={className}>
      <ButtonGroup
        className="w-full"
      >
        <NaturalLanguageDateTimeInput
          value={current}
          setValue={setValue}
          placeholder={placeholder}
        />

        <DateTimePicker24h
          value={current ?? undefined}
          onChange={(d) => setValue(d)}
        />
      </ButtonGroup>
      <p className="mt-1 text-xs text-muted-foreground">Asia/Bangkok (UTC+07:00)</p>
    </div>
  );
}
