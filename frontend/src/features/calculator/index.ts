export { Calculator, type CalculatorProps } from "@/features/calculator/components/Calculator";
export type { CalculatorHistoryProps } from "@/features/calculator/components/CalculatorHistory";
export {
  CalculatorKey,
  type CalculatorKeyProps,
  describeKey,
} from "@/features/calculator/components/CalculatorKey";
export {
  CalculatorKeypad,
  type CalculatorKeypadProps,
  KEYPAD_LAYOUT,
} from "@/features/calculator/components/CalculatorKeypad";
export { HEALTH_POLL_INTERVAL_MS } from "@/features/calculator/components/CalculatorStatus";
export {
  ApiClientProvider,
  type ApiClientProviderProps,
  useApiClient,
} from "@/features/calculator/context/ApiClientContext";
export {
  type CalculatorContextValue,
  CalculatorProvider,
  type CalculatorProviderProps,
  useCalculator,
} from "@/features/calculator/context/CalculatorContext";
export {
  binaryExpression,
  completedExpression,
  formatOperand,
  historyLine,
  pendingExpression,
  unaryExpression,
} from "@/features/calculator/state/formatting";
export {
  DIGITS,
  isEditableTarget,
  keyActionFromKeyboardEvent,
} from "@/features/calculator/state/keyboard";
export {
  calculatorReducer,
  createInitialState,
  HISTORY_LIMIT,
  INITIAL_INPUT,
  MAX_INPUT_LENGTH,
} from "@/features/calculator/state/reducer";
export type {
  CalculatorAction,
  CalculatorError,
  CalculatorState,
  CalculatorStatus,
  Digit,
  HistoryEntry,
  KeyAction,
} from "@/features/calculator/state/types";
