interface LogoProps {
  size?: number;
  containerClassName?: string;
  showBackground?: boolean;
}

export default function Logo({
  size = 60,
  containerClassName = "mb-8",
  showBackground = true,
}: LogoProps = {}) {
  return (
    <div className={`flex justify-center select-none ${containerClassName}`}>
      <div
        className="relative select-none"
        style={{ width: size, height: size }}
      >
        {showBackground && (
          <div className="absolute inset-0 rounded-full bg-white" />
        )}
        <img
          src="/susan.svg"
          width={size}
          height={size}
          alt="Susan"
          draggable={false}
          className={`relative block h-full w-full select-none ${
            showBackground ? "" : "dark:invert"
          }`}
        />
      </div>
    </div>
  );
}
